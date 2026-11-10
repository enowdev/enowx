package recovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReconcileResult reports the outcome of history validation and rollback.
type ReconcileResult struct {
	TotalLines     int
	ValidSteps     int
	TruncatedBytes int64
	LastStepIndex  int
	Recovered      bool
}

// ReconcileHistory inspects a JSONL history file, validates line-by-line JSON integrity,
// and truncates any partial or corrupted trailing line resulting from a crash or sudden termination.
func ReconcileHistory(filePath string) (*ReconcileResult, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			return &ReconcileResult{}, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var validOffset int64
	var currentOffset int64
	var totalLines int
	var validSteps int
	var lastStepIndex int = -1

	for {
		line, isPrefix, readErr := reader.ReadLine()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading history: %w", readErr)
		}

		lineLen := int64(len(line)) + 1 // +1 for newline delimiter
		if isPrefix {
			// Incomplete long buffer line, mark corruption
			break
		}

		// Verify valid JSON
		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err == nil {
			totalLines++
			validOffset = currentOffset + lineLen
			if idx, ok := raw["step_index"].(float64); ok {
				stepIdx := int(idx)
				if stepIdx > lastStepIndex {
					lastStepIndex = stepIdx
					validSteps++
				}
			}
		} else {
			// Line is corrupted / partial write at EOF
			break
		}

		currentOffset += lineLen
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	result := &ReconcileResult{
		TotalLines:    totalLines,
		ValidSteps:    validSteps,
		LastStepIndex: lastStepIndex,
	}

	if fi.Size() > validOffset {
		// Truncate corrupted trailing data
		result.TruncatedBytes = fi.Size() - validOffset
		if err := f.Truncate(validOffset); err != nil {
			return nil, fmt.Errorf("failed to truncate corrupted trailing history: %w", err)
		}
		result.Recovered = true
	}

	return result, nil
}

// SyntheticRestorationPrompt creates an injection prompt notifying the agent of checkpoint resumption.
func SyntheticRestorationPrompt(cp Checkpoint, failure *FailureContext) string {
	var reason string
	if failure != nil && failure.ErrorCode != "" {
		reason = fmt.Sprintf("Interruption reason: %s.", failure.ErrorCode)
		if failure.ErrorMessage != "" {
			reason += fmt.Sprintf(" (%s)", SanitizeText(failure.ErrorMessage))
		}
	} else {
		reason = "Interrupted by system disconnect."
	}

	return fmt.Sprintf("[System: Resuming session execution from Checkpoint Step #%d. %s Workspace state and history integrity verified. Please continue task from last successful transaction.]",
		cp.LastStepIndex, reason)
}
