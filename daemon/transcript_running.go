package daemon

import (
	"os"

	"github.com/enowx/enowx/core/agent"
	"github.com/enowx/enowx/core/procstat"
	"github.com/enowx/enowx/core/transcript"
)

// runningConversation is the conversation session id's agent says it is on, for an agent whose
// hooks can't say it (Claude Code on Windows). Without it a session whose agent went on with a
// conversation begun before the session, typed by hand or by /resume, showed nothing in its chat:
// that file reads as another session's. A variable, for tests to stand in for real processes.
var runningConversation = func(id, agentID, home string) string {
	if agent.Base(agentID) != "claude" || len(agent.HookArgs(agentID)) > 0 {
		return ""
	}
	return transcript.ClaudeRunning(transcript.ClaudeHome(home), sessionProcesses(id))
}

// sessionProcesses are the processes session id runs, from its root down, each level in turn.
func sessionProcesses(id string) []int {
	list, err := procstat.Snapshot()
	if err != nil {
		return nil
	}
	kids := make(map[int][]int, len(list))
	for _, p := range list {
		kids[p.PPID] = append(kids[p.PPID], p.PID)
	}
	var out, queue []int
	for _, root := range kids[os.Getpid()] {
		if sessionOf(root) == id {
			queue = append(queue, root)
		}
	}
	seen := map[int]bool{}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		out = append(out, pid)
		queue = append(queue, kids[pid]...)
	}
	return out
}
