package recovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockFileName = "session.lock"

type lockRecord struct {
	PID       int       `json:"pid"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

// AcquireLock attempts to acquire a file-based lock for the session.
// If a lock already exists, it checks if the holding process is still alive.
// If stale, it cleans up and acquires the lock. Returns an unlock callback.
func AcquireLock(stateDir, sessionID string) (func(), error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockPath := filepath.Join(stateDir, lockFileName)

	// Check existing lock
	if data, err := os.ReadFile(lockPath); err == nil {
		var existing lockRecord
		if json.Unmarshal(data, &existing) == nil && existing.PID > 0 {
			if isProcessAlive(existing.PID) && existing.PID != os.Getpid() {
				return nil, fmt.Errorf("%w: held by PID %d (session: %s)", ErrSessionLocked, existing.PID, existing.SessionID)
			}
			// Process is dead, stale lock can be cleaned up
			_ = os.Remove(lockPath)
		}
	}

	// Create new lock
	rec := lockRecord{
		PID:       os.Getpid(),
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
	recData, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(lockPath, recData, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write session lock: %w", err)
	}

	unlock := func() {
		_ = os.Remove(lockPath)
	}

	return unlock, nil
}

// IsLocked reports whether stateDir currently has an active lock held by a live process.
func IsLocked(stateDir string) bool {
	lockPath := filepath.Join(stateDir, lockFileName)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false
	}
	var existing lockRecord
	if err := json.Unmarshal(data, &existing); err != nil || existing.PID <= 0 {
		return false
	}
	return isProcessAlive(existing.PID)
}
