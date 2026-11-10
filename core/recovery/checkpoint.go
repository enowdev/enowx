package recovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const checkpointFileName = "checkpoint.json"

// ComputeStateHash calculates the SHA-256 hash of a file at snapshotPath.
func ComputeStateHash(snapshotPath string) (string, error) {
	if snapshotPath == "" {
		return "", nil
	}
	f, err := os.Open(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SaveCheckpoint writes the session state to checkpoint.json atomically.
func SaveCheckpoint(stateDir string, state *SessionState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Compute snapshot hash if ref exists
	if state.HistorySnapshotRef != "" {
		hash, err := ComputeStateHash(state.HistorySnapshotRef)
		if err == nil && hash != "" {
			state.Checkpoint.StateHash = hash
		}
	}

	// Sanitize failure context if present
	if state.FailureContext != nil {
		state.FailureContext = SanitizeFailureContext(state.FailureContext)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	targetPath := filepath.Join(stateDir, checkpointFileName)
	tmpPath := targetPath + ".tmp"

	// Write to temporary journal file
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write checkpoint temp file: %w", err)
	}

	// Atomic rename replaces the existing checkpoint safely
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically rename checkpoint file: %w", err)
	}

	return nil
}

// LoadCheckpoint reads and unmarshals checkpoint.json from stateDir.
func LoadCheckpoint(stateDir string) (*SessionState, error) {
	targetPath := filepath.Join(stateDir, checkpointFileName)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("%w: read error: %v", ErrSessionUnrecoverable, err)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("%w: json decode error: %v", ErrSessionUnrecoverable, err)
	}

	return &state, nil
}
