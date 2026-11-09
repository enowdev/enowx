package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// ClaudeRunning is the conversation one of pids is on, as Claude Code records it for each of
// its processes in claudeHome/sessions/<pid>.json, "" when none of them says. The first of pids
// that says wins: callers list a session's processes from its root down, so the agent comes
// before what it started. A conversation moved to by /resume shows there too, which is what
// says it when no hook does (Claude Code's hooks don't run on Windows).
func ClaudeRunning(claudeHome string, pids []int) string {
	for _, pid := range pids {
		b, err := os.ReadFile(filepath.Join(claudeHome, "sessions", strconv.Itoa(pid)+".json"))
		if err != nil {
			continue
		}
		var rec struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(b, &rec) != nil || rec.PID != pid || rec.SessionID == "" {
			continue // half written, or not this process's
		}
		return rec.SessionID
	}
	return ""
}
