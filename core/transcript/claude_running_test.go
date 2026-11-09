package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

// Claude Code says, for each of its processes, the conversation it is on: the first of a
// session's processes that says it names the session's.
func TestClaudeRunningReadsWhatTheProcessSays(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("20.json", `{"pid":20,"sessionId":"the-agents"}`)
	write("30.json", `{"pid":30,"sessionId":"a-child-claude"}`)
	write("40.json", `{"pid":41,"sessionId":"another-process"}`)
	write("50.json", `{"pid":50,"sess`)

	cases := []struct {
		pids []int
		want string
	}{
		{[]int{10, 20, 30}, "the-agents"}, // the wrapper says nothing; the agent comes before its children
		{[]int{30, 20}, "a-child-claude"},
		{[]int{40}, ""}, // a file naming another process
		{[]int{50}, ""}, // half written
		{[]int{60}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := ClaudeRunning(home, c.pids); got != c.want {
			t.Errorf("ClaudeRunning(%v) = %q, want %q", c.pids, got, c.want)
		}
	}
}
