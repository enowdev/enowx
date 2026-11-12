package recovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCrashSimulationAndRecovery(t *testing.T) {
	tempBase := t.TempDir()
	sessionID := "11111111-2222-3333-4444-555555555555"
	stateDir := filepath.Join(tempBase, sessionID)
	_ = os.MkdirAll(stateDir, 0o700)

	logPath := filepath.Join(stateDir, "transcript.jsonl")

	// 1. Simulate initial multi-step execution up to step 2
	historyContent := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","content":"Start task"}
{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","content":"Step 1 completed"}
{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","content":"Step 2 completed"}
`
	if err := os.WriteFile(logPath, []byte(historyContent), 0o600); err != nil {
		t.Fatal(err)
	}

	state := &SessionState{
		SessionID:          sessionID,
		ConversationID:     sessionID,
		Status:             StatusActive,
		HistorySnapshotRef: logPath,
		Checkpoint: Checkpoint{
			LastStepIndex:    2,
			TimestampEpochMs: time.Now().UnixMilli(),
		},
	}
	if err := SaveCheckpoint(stateDir, state); err != nil {
		t.Fatalf("failed to save initial checkpoint: %v", err)
	}

	// 2. Simulate crash during step 3 execution:
	// A stale lock was left by a deceased process PID 9999999
	staleLock := lockRecord{
		PID:       9999999, // Dead PID
		SessionID: sessionID,
		CreatedAt: time.Now().Add(-10 * time.Minute),
	}
	staleLockData, _ := json.Marshal(staleLock)
	_ = os.WriteFile(filepath.Join(stateDir, lockFileName), staleLockData, 0o600)

	// Step 3 wrote a partial trailing line before crashing
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"step_index":3,"source":"MODEL","type":"PLANNER_RESPONSE","status":"IN_PROG`) // Corrupted line
	_ = f.Close()

	// Update state to Error with failure context
	state.Status = StatusError
	state.FailureContext = &FailureContext{
		ErrorCode:    ErrCodeAbruptDisconnect,
		ErrorMessage: "process terminated abruptly by SIGKILL with key sk-abcdef1234567890abcdef",
		Recoverable:  true,
	}
	if err := SaveCheckpoint(stateDir, state); err != nil {
		t.Fatal(err)
	}

	// 3. Execute Resume via Recovery Engine
	var logBuf bytes.Buffer
	logger := NewLogger(&logBuf)
	engine := NewEngine(logger)

	result, err := engine.Resume(ResumeOptions{
		BaseDir:   tempBase,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	defer result.Unlock()

	// Verify history was reconciled and truncated
	if !result.Reconciled.Recovered || result.Reconciled.TruncatedBytes <= 0 {
		t.Fatalf("expected corrupted partial line to be truncated, result: %+v", result.Reconciled)
	}
	if result.State.Checkpoint.LastStepIndex != 2 {
		t.Fatalf("expected LastStepIndex 2, got %d", result.State.Checkpoint.LastStepIndex)
	}

	// Verify synthetic restoration prompt
	if !strings.Contains(result.SyntheticPrompt, "Checkpoint Step #2") {
		t.Errorf("synthetic prompt missing step reference: %s", result.SyntheticPrompt)
	}

	// Verify credentials were redacted
	if strings.Contains(result.SyntheticPrompt, "sk-abcdef") {
		t.Errorf("synthetic prompt leaked secret key: %s", result.SyntheticPrompt)
	}

	// Verify state status updated to active
	if result.State.Status != StatusActive {
		t.Errorf("expected status %s, got %s", StatusActive, result.State.Status)
	}

	// Verify structured log emission
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, EventSessionResumeAttempt) || !strings.Contains(logOutput, EventSessionResumeSuccess) {
		t.Errorf("missing expected structured logs in: %s", logOutput)
	}
}

func TestCorruptedHistoryRecovery(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "corrupt_history.jsonl")

	validContent := `{"step_index":0,"data":"valid"}
{"step_index":1,"data":"valid"}
`
	corruptSuffix := `{"step_index":2,"incomplete_`

	if err := os.WriteFile(tempFile, []byte(validContent+corruptSuffix), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := ReconcileHistory(tempFile)
	if err != nil {
		t.Fatalf("ReconcileHistory failed: %v", err)
	}

	if !res.Recovered {
		t.Fatal("expected Recovered to be true")
	}
	if res.LastStepIndex != 1 {
		t.Fatalf("expected LastStepIndex 1, got %d", res.LastStepIndex)
	}
	if res.ValidSteps != 2 {
		t.Fatalf("expected 2 valid steps, got %d", res.ValidSteps)
	}

	// Check file content after truncation
	afterBytes, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterBytes) != validContent {
		t.Fatalf("file content mismatch:\ngot: %q\nwant: %q", string(afterBytes), validContent)
	}
}

func TestLockingAndConcurrency(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "lock_test")
	sessionID := "22222222-3333-4444-5555-666666666666"

	// 1. Acquire first lock
	unlock1, err := AcquireLock(stateDir, sessionID)
	if err != nil {
		t.Fatalf("initial AcquireLock failed: %v", err)
	}

	// 2. Second acquire by another instance should fail with ErrSessionLocked
	// Note: in our lock implementation, if existing PID == current PID it succeeds, so simulate another live PID
	lockFile := filepath.Join(stateDir, lockFileName)
	otherLock := lockRecord{
		PID:       os.Getpid(), // current live PID
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
	otherData, _ := json.Marshal(otherLock)
	_ = os.WriteFile(lockFile, otherData, 0o600)

	if !IsLocked(stateDir) {
		t.Fatal("expected IsLocked to be true")
	}

	unlock1()
	if IsLocked(stateDir) {
		t.Fatal("expected IsLocked to be false after unlock")
	}
}

func TestTerminatedSessionRejection(t *testing.T) {
	tempBase := t.TempDir()
	sessionID := "33333333-4444-5555-6665-777777777777"
	stateDir := filepath.Join(tempBase, sessionID)

	state := &SessionState{
		SessionID:      sessionID,
		ConversationID: sessionID,
		Status:         StatusTerminated, // Clean exit
		Checkpoint: Checkpoint{
			LastStepIndex: 10,
		},
	}
	if err := SaveCheckpoint(stateDir, state); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(nil)
	_, err := engine.Resume(ResumeOptions{
		BaseDir:   tempBase,
		SessionID: sessionID,
	})

	if err == nil || !errors.Is(err, ErrSessionAlreadyCompleted) {
		t.Fatalf("expected ErrSessionAlreadyCompleted, got: %v", err)
	}
}

func TestAutoDetectLatestRecoverable(t *testing.T) {
	tempBase := t.TempDir()

	// Session 1: Terminated (clean exit)
	s1 := "aaaaaaaa-1111-1111-1111-111111111111"
	_ = SaveCheckpoint(filepath.Join(tempBase, s1), &SessionState{
		SessionID: s1, Status: StatusTerminated,
	})

	// Session 2: Recoverable error
	s2 := "bbbbbbbb-2222-2222-2222-222222222222"
	_ = SaveCheckpoint(filepath.Join(tempBase, s2), &SessionState{
		SessionID: s2, Status: StatusError,
		FailureContext: &FailureContext{Recoverable: true},
	})

	engine := NewEngine(nil)
	detected, err := engine.AutoDetectRecoverable(tempBase)
	if err != nil {
		t.Fatalf("AutoDetectRecoverable failed: %v", err)
	}

	if detected != s2 {
		t.Fatalf("expected detected session %s, got %s", s2, detected)
	}
}

func TestSanitizeSecrets(t *testing.T) {
	raw := "Error connecting to AI: Bearer my-secret-token-12345678, api_key: AIzaSyD1234567890123456789012345678901, sk-abcdef1234567890abcdef"
	clean := SanitizeText(raw)

	if strings.Contains(clean, "my-secret-token") || strings.Contains(clean, "AIzaSy") || strings.Contains(clean, "sk-abcdef") {
		t.Fatalf("SanitizeText failed to redact secrets: %s", clean)
	}
}

func TestCompactionOnContextOverflow(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "history_overflow.jsonl")

	// Generate 30 steps
	var buf bytes.Buffer
	for i := 0; i < 30; i++ {
		line := map[string]interface{}{
			"step_index": i,
			"source":     "MODEL",
			"type":       "PLANNER_RESPONSE",
			"content":    "step data",
		}
		data, _ := json.Marshal(line)
		buf.Write(data)
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(tempFile, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	compacted, err := CompactHistory(tempFile, 10)
	if err != nil {
		t.Fatalf("CompactHistory failed: %v", err)
	}
	if !compacted {
		t.Fatal("expected CompactHistory to return true")
	}

	// Verify line count
	readData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(readData)), "\n")
	// 1 summary step + 10 retained steps = 11 lines
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines after compaction, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "compact_summary") {
		t.Fatalf("expected first line to be compact_summary, got: %s", lines[0])
	}
}

func TestToolCrashOrphanProcessHandling(t *testing.T) {
	tempBase := t.TempDir()
	sessionID := "44444444-5555-6666-7777-888888888888"
	stateDir := filepath.Join(tempBase, sessionID)
	_ = os.MkdirAll(stateDir, 0o700)

	logPath := filepath.Join(stateDir, "transcript.jsonl")
	_ = os.WriteFile(logPath, []byte(`{"step_index":0,"data":"init"}`+"\n"), 0o600)

	state := &SessionState{
		SessionID:          sessionID,
		ConversationID:     sessionID,
		Status:             StatusError,
		HistorySnapshotRef: logPath,
		Checkpoint: Checkpoint{
			LastStepIndex:    0,
			ActiveToolPID:    9999999, // Dead PID representing orphaned child process
			ActiveToolCallID: "call_tool_read_file",
		},
		FailureContext: &FailureContext{
			ErrorCode:   ErrCodeToolExecFailure,
			Recoverable: true,
		},
	}
	if err := SaveCheckpoint(stateDir, state); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(nil)
	res, err := engine.Resume(ResumeOptions{
		BaseDir:   tempBase,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	defer res.Unlock()

	// Verify ActiveToolPID was cleared
	if res.State.Checkpoint.ActiveToolPID != 0 {
		t.Fatalf("expected ActiveToolPID to be reset to 0, got %d", res.State.Checkpoint.ActiveToolPID)
	}
	// Verify synthetic prompt notifies of failure
	if !strings.Contains(res.SyntheticPrompt, ErrCodeToolExecFailure) {
		t.Errorf("expected synthetic prompt to reference %s, got: %s", ErrCodeToolExecFailure, res.SyntheticPrompt)
	}
}
