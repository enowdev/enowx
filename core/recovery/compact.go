package recovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// HistoryStep represents a generic step entry in a session transcript/history.
type HistoryStep struct {
	StepIndex int                    `json:"step_index"`
	Source    string                 `json:"source,omitempty"`
	Type      string                 `json:"type,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Content   string                 `json:"content,omitempty"`
	CreatedAt string                 `json:"created_at,omitempty"`
	Raw       map[string]interface{} `json:"-"`
}

// CompactHistory compacts a JSONL history file to prevent context overflow.
// It summarizes older steps and retains the most recent retainSteps steps intact.
func CompactHistory(filePath string, retainSteps int) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("compact: open history file: %w", err)
	}

	var allLines []map[string]interface{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err == nil {
			allLines = append(allLines, entry)
		}
	}
	scanErr := scanner.Err()
	_ = f.Close()
	if scanErr != nil {
		return false, fmt.Errorf("compact: read history: %w", scanErr)
	}

	if len(allLines) <= retainSteps {
		return false, nil // No compaction needed
	}

	cutoff := len(allLines) - retainSteps
	compactedCount := cutoff

	// Create summary of compacted history
	summaryContent := fmt.Sprintf("[Compacted context: %d earlier interaction steps summarized to fit context budget]", compactedCount)
	summaryStep := map[string]interface{}{
		"step_index": 0,
		"type":       "compact_summary",
		"source":     "SYSTEM",
		"status":     "DONE",
		"content":    summaryContent,
	}

	newLines := make([]map[string]interface{}, 0, retainSteps+1)
	newLines = append(newLines, summaryStep)
	newLines = append(newLines, allLines[cutoff:]...)

	// Atomic write back
	tmpPath := filePath + ".compact.tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return false, fmt.Errorf("compact: create tmp file: %w", err)
	}

	writer := bufio.NewWriter(tmpFile)
	for _, l := range newLines {
		data, err := json.Marshal(l)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return false, fmt.Errorf("compact: marshal line: %w", err)
		}
		if _, err := writer.Write(data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return false, fmt.Errorf("compact: write line: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return false, fmt.Errorf("compact: write newline: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return false, fmt.Errorf("compact: flush: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return false, fmt.Errorf("compact: sync: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, filePath); err != nil {
		// Fallback for Windows if target exists
		_ = os.Remove(filePath)
		if rerr := os.Rename(tmpPath, filePath); rerr != nil {
			os.Remove(tmpPath)
			return false, fmt.Errorf("compact: rename tmp file: %w", rerr)
		}
	}

	return true, nil
}
