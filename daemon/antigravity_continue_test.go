package daemon

import (
	"testing"
	"time"

	"github.com/enowx/enowx/core/session"
)

func TestContinueAntigravityOlderSession(t *testing.T) {
	srv := startServer(t)

	// Simulate an older Antigravity session loaded from sessions.json where Relaunch was false
	id := "test-ag-older-session"
	spec := session.Spec{
		ID:      id,
		Dir:     t.TempDir(),
		Created: time.Now().Add(-1 * time.Hour),
		Command: []string{"agy", "--dangerously-skip-permissions"},
		Exited:  true,
		Meta: session.Meta{
			Agent:           "antigravity",
			Relaunch:        false, // older session before continue flag was added
			SkipPermissions: true,
		},
	}
	sess, err := srv.sessions.Create(spec)
	if err != nil {
		t.Fatalf("Create session failed: %v", err)
	}

	// resumeSessionIn should succeed and not return "this session's command can't be continued"
	next, err := srv.resumeSessionIn(sess.ID(), "", "")
	if err != nil {
		t.Fatalf("resumeSessionIn failed: %v", err)
	}
	if next == nil {
		t.Fatal("expected relaunched session, got nil")
	}
	if !next.Meta().Relaunch {
		t.Errorf("expected Relaunch to be true on recovered session")
	}
}
