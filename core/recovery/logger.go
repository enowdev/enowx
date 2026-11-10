package recovery

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// Standard session recovery event names.
const (
	EventSessionInitialized   = "SESSION_INITIALIZED"
	EventCheckpointSaved      = "CHECKPOINT_SAVED"
	EventSessionCrashDetected = "SESSION_CRASH_DETECTED"
	EventSessionResumeAttempt = "SESSION_RESUME_ATTEMPT"
	EventSessionResumeSuccess = "SESSION_RESUME_SUCCESS"
	EventSessionResumeFailed  = "SESSION_RESUME_FAILED"
)

// StructuredLogRecord represents an observability log entry for recovery.
type StructuredLogRecord struct {
	Timestamp            string                 `json:"timestamp"`
	Event                string                 `json:"event"`
	SessionID            string                 `json:"session_id"`
	StepIndex            int                    `json:"step_index"`
	DurationMs           int64                  `json:"duration_ms"`
	RecoveryAttemptCount int                    `json:"recovery_attempt_count"`
	Details              map[string]interface{} `json:"details,omitempty"`
}

// Logger writes structured JSON recovery logs.
type Logger struct {
	mu  sync.Mutex
	out io.Writer
}

// NewLogger creates a new structured recovery logger.
func NewLogger(out io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{out: out}
}

var defaultLogger = NewLogger(os.Stdout)

// SetDefaultOutput redirects the global recovery logger output.
func SetDefaultOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.out = w
}

// Log emits a structured recovery event.
func (l *Logger) Log(event, sessionID string, stepIndex int, duration time.Duration, attemptCount int, details map[string]interface{}) {
	record := StructuredLogRecord{
		Timestamp:            time.Now().UTC().Format(time.RFC3339Nano),
		Event:                event,
		SessionID:            sessionID,
		StepIndex:            stepIndex,
		DurationMs:           duration.Milliseconds(),
		RecoveryAttemptCount: attemptCount,
		Details:              details,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.out.Write(append(data, '\n'))
}

// EmitLog logs an event using the default recovery logger.
func EmitLog(event, sessionID string, stepIndex int, duration time.Duration, attemptCount int, details map[string]interface{}) {
	defaultLogger.Log(event, sessionID, stepIndex, duration, attemptCount, details)
}
