package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ResumeOptions sets arguments for a resume operation.
type ResumeOptions struct {
	BaseDir      string // Directory containing session folders (e.g. ~/.enowx-app/sessions)
	SessionID    string // Optional: if empty, auto-detect latest recoverable session
	MaxAttempts  int
	AttemptCount int
}

// ResumeResult contains hydrated recovery context and active locks.
type ResumeResult struct {
	State           *SessionState
	SyntheticPrompt string
	Unlock          func()
	Reconciled      *ReconcileResult
}

// Engine orchestrates session resumption and state hydration.
type Engine struct {
	logger *Logger
}

// NewEngine creates a recovery Engine.
func NewEngine(l *Logger) *Engine {
	if l == nil {
		l = defaultLogger
	}
	return &Engine{logger: l}
}

// Resume executes the full recovery sequence for a broken session.
func (e *Engine) Resume(opts ResumeOptions) (*ResumeResult, error) {
	start := time.Now()
	sessionID := strings.TrimSpace(opts.SessionID)

	// Auto-detect latest recoverable session if sessionID not provided
	if sessionID == "" {
		detectedID, err := e.AutoDetectRecoverable(opts.BaseDir)
		if err != nil {
			return nil, err
		}
		sessionID = detectedID
	}

	// Validate UUID format
	if !uuidRegex.MatchString(sessionID) {
		return nil, fmt.Errorf("malformed session ID: %q must be a valid UUID v4", sessionID)
	}

	stateDir := filepath.Join(opts.BaseDir, sessionID)

	e.logger.Log(EventSessionResumeAttempt, sessionID, 0, 0, opts.AttemptCount, map[string]interface{}{
		"state_dir": stateDir,
	})

	// 1. Acquire Lock
	unlock, err := AcquireLock(stateDir, sessionID)
	if err != nil {
		e.logger.Log(EventSessionResumeFailed, sessionID, 0, time.Since(start), opts.AttemptCount, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// 2. Load Checkpoint
	state, err := LoadCheckpoint(stateDir)
	if err != nil {
		unlock()
		e.logger.Log(EventSessionResumeFailed, sessionID, 0, time.Since(start), opts.AttemptCount, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// 3. Status Validation
	if state.Status == StatusTerminated {
		unlock()
		err := fmt.Errorf("%w (session %s has clean exit)", ErrSessionAlreadyCompleted, sessionID)
		e.logger.Log(EventSessionResumeFailed, sessionID, state.Checkpoint.LastStepIndex, time.Since(start), opts.AttemptCount, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// 4. Reconcile History (Rollback corrupted partial writes)
	var recon *ReconcileResult
	if state.HistorySnapshotRef != "" {
		res, err := ReconcileHistory(state.HistorySnapshotRef)
		if err != nil {
			unlock()
			e.logger.Log(EventSessionResumeFailed, sessionID, state.Checkpoint.LastStepIndex, time.Since(start), opts.AttemptCount, map[string]interface{}{
				"error": err.Error(),
			})
			return nil, fmt.Errorf("history reconciliation error: %w", err)
		}
		recon = res
		if recon.LastStepIndex >= 0 {
			state.Checkpoint.LastStepIndex = recon.LastStepIndex
		}

		// Safe compaction if context overflowed
		if state.FailureContext != nil && state.FailureContext.ErrorCode == ErrCodeContextOverflow {
			if _, cerr := CompactHistory(state.HistorySnapshotRef, 20); cerr != nil {
				e.logger.Log(EventSessionResumeAttempt, sessionID, state.Checkpoint.LastStepIndex, 0, opts.AttemptCount, map[string]interface{}{
					"compaction_warning": cerr.Error(),
				})
			}
		}
	}

	// 5. Handle Tool Crash & Orphan Process Cleanup
	if state.Checkpoint.ActiveToolPID > 0 {
		if isProcessAlive(state.Checkpoint.ActiveToolPID) {
			_ = terminateProcess(state.Checkpoint.ActiveToolPID)
		}
		if state.FailureContext == nil {
			state.FailureContext = &FailureContext{
				ErrorCode:    ErrCodeToolExecFailure,
				ErrorMessage: fmt.Sprintf("Tool call %s terminated due to process crash", state.Checkpoint.ActiveToolCallID),
				Recoverable:  true,
			}
		}
		state.Checkpoint.ActiveToolPID = 0
	}

	// 6. Generate Synthetic Restoration Prompt
	prompt := SyntheticRestorationPrompt(state.Checkpoint, state.FailureContext)

	// 7. Update Status
	state.Status = StatusActive
	_ = SaveCheckpoint(stateDir, state)

	var truncatedBytes int64
	if recon != nil {
		truncatedBytes = recon.TruncatedBytes
	}
	e.logger.Log(EventSessionResumeSuccess, sessionID, state.Checkpoint.LastStepIndex, time.Since(start), opts.AttemptCount, map[string]interface{}{
		"truncated_bytes": truncatedBytes,
		"last_step":       state.Checkpoint.LastStepIndex,
	})

	return &ResumeResult{
		State:           state,
		SyntheticPrompt: prompt,
		Unlock:          unlock,
		Reconciled:      recon,
	}, nil
}

// AutoDetectRecoverable finds the newest session directory marked recoverable.
func (e *Engine) AutoDetectRecoverable(baseDir string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to read base directory %q: %w", baseDir, err)
	}

	var bestID string
	var latestMod time.Time

	for _, entry := range entries {
		if !entry.IsDir() || !uuidRegex.MatchString(entry.Name()) {
			continue
		}
		sessDir := filepath.Join(baseDir, entry.Name())
		state, err := LoadCheckpoint(sessDir)
		if err != nil {
			continue
		}
		if state.Status != StatusTerminated && (state.FailureContext == nil || state.FailureContext.Recoverable) {
			if fi, err := entry.Info(); err == nil {
				if fi.ModTime().After(latestMod) {
					latestMod = fi.ModTime()
					bestID = entry.Name()
				}
			}
		}
	}

	if bestID == "" {
		return "", fmt.Errorf("%w: no recoverable broken sessions found in %s", ErrSessionNotFound, baseDir)
	}

	return bestID, nil
}
