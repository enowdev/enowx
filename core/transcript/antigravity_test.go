package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/enowx/enowx/proto"
)

func TestAntigravityConversationID(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{
			path: "C:\\Users\\test\\.gemini\\antigravity-cli\\brain\\95f17248-01cd-432c-8dcd-ac35f2180110\\.system_generated\\logs\\transcript.jsonl",
			want: "95f17248-01cd-432c-8dcd-ac35f2180110",
		},
		{
			path: "/home/user/.gemini/antigravity-cli/brain/12345678-1234-1234-1234-123456789abc/.system_generated/logs/transcript.jsonl",
			want: "12345678-1234-1234-1234-123456789abc",
		},
		{
			path: "invalid/path/without/uuid.jsonl",
			want: "",
		},
	}

	for _, tc := range cases {
		got := AntigravityConversationID(tc.path)
		if got != tc.want {
			t.Errorf("AntigravityConversationID(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestAntigravityReaderParsing(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "transcript.jsonl")

	content := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","created_at":"2026-11-10T03:30:38Z","content":"<USER_REQUEST>\ngit pull\n</USER_REQUEST>\n<ADDITIONAL_METADATA>\ninfo\n</ADDITIONAL_METADATA>"}
{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-11-10T03:30:39Z","input_tokens":100,"output_tokens":50,"tool_calls":[{"name":"run_command","args":{"CommandLine":"git pull"}}]}
{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-11-10T03:30:44Z","content":"Pull completed successfully"}
`
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewAntigravityReader(logPath)
	if err := reader.Update(); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	entries, _ := reader.Since(0)

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d: %+v", len(entries), entries)
	}

	// Verify User input extraction
	if entries[0].Kind != proto.TranscriptUser || entries[0].Text != "git pull" {
		t.Errorf("entry 0: kind=%v text=%q", entries[0].Kind, entries[0].Text)
	}

	// Verify Assistant Tool Call
	if entries[1].Kind != proto.TranscriptTool || entries[1].Text != "run_command" {
		t.Errorf("entry 1: kind=%v text=%q", entries[1].Kind, entries[1].Text)
	}

	// Verify Assistant Message
	if entries[2].Kind != proto.TranscriptAssistant || entries[2].Text != "Pull completed successfully" {
		t.Errorf("entry 2: kind=%v text=%q", entries[2].Kind, entries[2].Text)
	}

	// Verify Token calculation
	if reader.Context != 150 {
		t.Errorf("reader.Context = %d, want 150", reader.Context)
	}
}

func TestAntigravityReaderToleratesCorruptedTrailingLine(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "transcript.jsonl")

	// Corrupted line at the end (simulating abrupt kill mid-write)
	content := `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","created_at":"2026-11-10T03:30:38Z","content":"<USER_REQUEST>\ncheck status\n</USER_REQUEST>"}
{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DO
`
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	reader := NewAntigravityReader(logPath)
	if err := reader.Update(); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	entries, _ := reader.Since(0)

	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry despite corrupted EOF line, got %d", len(entries))
	}
	if entries[0].Text != "check status" {
		t.Errorf("expected text 'check status', got %q", entries[0].Text)
	}
}

func TestAntigravityFilesDiscovery(t *testing.T) {
	home := t.TempDir()
	geminiDir := filepath.Join(home, ".gemini", "antigravity-cli")
	_ = os.MkdirAll(geminiDir, 0o700)

	convoID := "88888888-4444-4444-4444-121212121212"
	brainLogDir := filepath.Join(geminiDir, "brain", convoID, ".system_generated", "logs")
	_ = os.MkdirAll(brainLogDir, 0o700)
	tFile := filepath.Join(brainLogDir, "transcript.jsonl")
	_ = os.WriteFile(tFile, []byte("{}\n"), 0o600)

	otherConvoID := "99999999-4444-4444-4444-121212121212"
	otherLogDir := filepath.Join(geminiDir, "brain", otherConvoID, ".system_generated", "logs")
	_ = os.MkdirAll(otherLogDir, 0o700)
	otherTFile := filepath.Join(otherLogDir, "transcript.jsonl")
	_ = os.WriteFile(otherTFile, []byte("{}\n"), 0o600)

	ws := filepath.Join(home, "myproject")
	otherWs := filepath.Join(home, "otherproject")

	rec1, _ := json.Marshal(map[string]any{"workspace": ws, "conversationId": convoID})
	rec2, _ := json.Marshal(map[string]any{"workspace": otherWs, "conversationId": otherConvoID})
	_ = os.WriteFile(filepath.Join(geminiDir, "history.jsonl"), []byte(string(rec1)+"\n"+string(rec2)+"\n"), 0o600)

	// Matching workspace returns its transcript
	found := AntigravityFiles(home, ws)
	if len(found) != 1 {
		t.Fatalf("expected 1 found transcript for matching workspace, got %d", len(found))
	}
	if found[0].Path != tFile {
		t.Errorf("found path = %q, want %q", found[0].Path, tFile)
	}

	// Non-matching workspace returns empty
	emptyWs := filepath.Join(home, "nonexistent")
	foundEmpty := AntigravityFiles(home, emptyWs)
	if len(foundEmpty) != 0 {
		t.Fatalf("expected 0 transcripts for nonexistent workspace, got %d", len(foundEmpty))
	}
}
