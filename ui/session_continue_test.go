package ui

import (
	"testing"

	"github.com/enowx/enowx/core/agent"
	"github.com/enowx/enowx/proto"
)

func TestCanContinueAntigravity(t *testing.T) {
	// Exited Antigravity session should be eligible for continue
	sess := proto.SessionInfo{
		ID:       "sess-ag-123",
		Agent:    "antigravity",
		Exited:   true,
		ExitCode: 1,
		ExitWhy:  "process terminated abruptly",
	}
	if !canContinue(sess) {
		t.Errorf("expected canContinue to be true for exited antigravity session")
	}

	// Active session cannot be continued
	activeSess := proto.SessionInfo{
		ID:     "sess-ag-456",
		Agent:  "antigravity",
		Exited: false,
	}
	if canContinue(activeSess) {
		t.Errorf("expected canContinue to be false for active session")
	}

	// Custom command session cannot be continued via agent continue
	customSess := proto.SessionInfo{
		ID:     "sess-ag-789",
		Agent:  agent.Custom,
		Exited: true,
	}
	if canContinue(customSess) {
		t.Errorf("expected canContinue to be false for custom command")
	}
}
