package recovery

import (
	"errors"
)

// Standard session statuses.
const (
	StatusActive     = "active"
	StatusPaused     = "paused"
	StatusError      = "error"
	StatusTerminated = "terminated"
)

// Standard failure error codes.
const (
	ErrCodeAbruptDisconnect = "SESSION_ABRUPT_DISCONNECT"
	ErrCodeToolExecFailure  = "TOOL_EXEC_FAILURE"
	ErrCodeContextOverflow  = "CONTEXT_OVERFLOW"
)

// Standard recovery error definitions.
var (
	ErrSessionNotFound         = errors.New("ERR_SESSION_NOT_FOUND: session ID not found")
	ErrSessionLocked           = errors.New("ERR_SESSION_LOCKED: session is currently active or locked by another instance")
	ErrSessionUnrecoverable    = errors.New("ERR_SESSION_UNRECOVERABLE: checkpoint metadata corrupted or invalid")
	ErrSessionAlreadyCompleted = errors.New("ERR_SESSION_ALREADY_COMPLETED: session completed normally and cannot be resumed")
)

// Checkpoint records the transactional state boundary of an agent execution.
type Checkpoint struct {
	LastStepIndex            int    `json:"last_step_index"`
	LastSuccessfulToolCallID string `json:"last_successful_tool_call_id,omitempty"`
	ActiveToolCallID         string `json:"active_tool_call_id,omitempty"`
	ActiveToolPID            int    `json:"active_tool_pid,omitempty"`
	TimestampEpochMs         int64  `json:"timestamp_epoch_ms"`
	StateHash                string `json:"state_hash"`
	UncommittedChanges       bool   `json:"uncommitted_changes"`
}

// FailureContext provides error details and recovery suitability.
type FailureContext struct {
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	StackTrace   string `json:"stack_trace,omitempty"`
	Recoverable  bool   `json:"recoverable"`
}

// SessionState is the persistent metadata schema for a resilient session.
type SessionState struct {
	SessionID          string          `json:"session_id"`
	ConversationID     string          `json:"conversation_id"`
	Status             string          `json:"status"`
	Checkpoint         Checkpoint      `json:"checkpoint"`
	FailureContext     *FailureContext `json:"failure_context,omitempty"`
	HistorySnapshotRef string          `json:"history_snapshot_ref,omitempty"`
}
