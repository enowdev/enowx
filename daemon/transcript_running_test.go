package daemon

import (
	"testing"
	"time"

	"github.com/enowx/enowx/core/session"
)

// An agent without hooks (Claude Code on Windows) that goes on with a conversation begun before
// its session, resumed by hand or by /resume, shows it in the chat: its process says it is on
// it. Without that word, such a conversation still reads as another session's.
func TestChatFollowsTheConversationItsAgentProcessRuns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	running := ""
	old := runningConversation
	runningConversation = func(id, agentID, home string) string { return running }
	t.Cleanup(func() { runningConversation = old })

	srv := startServer(t)
	dir := t.TempDir()
	sess, err := srv.sessions.Create(session.Spec{Command: []string{"sleep", "30"}, Dir: dir, Meta: session.Meta{Agent: "claude"}})
	if err != nil {
		t.Fatal(err)
	}
	earlier := claudeConversation(t, home, dir, "74fdbd63", time.Now().Add(-12*time.Hour), "from last night")

	if got := srv.transcriptPath(sess.ID()); got != "" {
		t.Fatalf("with nothing saying it is its own, it follows %q", got)
	}
	running = "74fdbd63"
	srv.transcripts.forget(sess.ID())
	if got := srv.transcriptPath(sess.ID()); got != earlier {
		t.Fatalf("its agent runs 74fdbd63, it follows %q", got)
	}
	if convo := sess.Meta().Conversation; convo != "74fdbd63" {
		t.Fatalf("kept %q: a restart wouldn't go on with it", convo)
	}
}
