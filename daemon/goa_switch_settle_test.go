package daemon

import (
	"testing"
	"time"
)

// looks plays back what a starting session shows, one look at a time, the last one after.
func looks(seq ...startLook) (func() startLook, *int) {
	n := 0
	return func() startLook {
		l := seq[min(n, len(seq)-1)]
		n++
		return l
	}, &n
}

// PowerShell on Windows runs alone for over a second before it starts the agent it wraps:
// that isn't the agent ending.
func TestGOASwitchWaitsForAWrapperToStartItsAgent(t *testing.T) {
	look, _ := looks(agentNotRunning, agentNotRunning, agentNotRunning, agentNotRunning, agentNotRunning, agentRunning)
	if !agentTakes(300*time.Millisecond, 10*time.Millisecond, look) {
		t.Fatal("an agent its wrapper hadn't started yet was taken as ended")
	}
}

// An agent seen running that then ends didn't take, and that is known at once.
func TestGOASwitchAgentSeenThenGoneFails(t *testing.T) {
	look, n := looks(agentNotRunning, agentRunning, agentRunning, agentNotRunning)
	start := time.Now()
	if agentTakes(5*time.Second, 10*time.Millisecond, look) {
		t.Fatal("an agent that ended was taken as started")
	}
	if took := time.Since(start); took > time.Second || *n != 4 {
		t.Fatalf("known after %d looks, %s", *n, took)
	}
}

// An agent never seen, one ending between two looks say, fails once the wait is up.
func TestGOASwitchAgentNeverSeenFailsWhenSettleIsUp(t *testing.T) {
	look, _ := looks(agentNotRunning)
	start := time.Now()
	if agentTakes(200*time.Millisecond, 10*time.Millisecond, look) {
		t.Fatal("an agent never seen was taken as started")
	}
	if took := time.Since(start); took < 200*time.Millisecond {
		t.Fatalf("gave up after %s, before the wait was up", took)
	}
}

// A session that has itself ended fails at once.
func TestGOASwitchSessionExitedFails(t *testing.T) {
	look, n := looks(agentNotRunning, agentExited)
	if agentTakes(5*time.Second, 10*time.Millisecond, look) || *n != 2 {
		t.Fatalf("an ended session: %d looks", *n)
	}
}

// An agent that starts and stays takes.
func TestGOASwitchAgentThatStaysTakes(t *testing.T) {
	look, _ := looks(agentRunning)
	if !agentTakes(100*time.Millisecond, 10*time.Millisecond, look) {
		t.Fatal("an agent that stayed up was taken as ended")
	}
}
