package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/enowx/enowx/proto"
)

var (
	antigravityUUIDRegex = regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)
	userReqOpen          = "<USER_REQUEST>"
	userReqClose         = "</USER_REQUEST>"
)

// AntigravityFiles returns the transcripts Antigravity keeps for sessions in dir, newest first.
func AntigravityFiles(home, dir string) []Found {
	cleanDir := filepath.Clean(dir)
	var all []Found
	seen := map[string]bool{}

	// 1. First check history.jsonl which maps workspace to conversationId
	historyPath := filepath.Join(home, ".gemini", "antigravity-cli", "history.jsonl")
	if f, err := os.Open(historyPath); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var rec struct {
				Workspace      string `json:"workspace"`
				ConversationID string `json:"conversationId"`
			}
			if json.Unmarshal(scanner.Bytes(), &rec) == nil && rec.ConversationID != "" {
				if strings.EqualFold(filepath.Clean(rec.Workspace), cleanDir) {
					tPath := filepath.Join(home, ".gemini", "antigravity-cli", "brain", rec.ConversationID, ".system_generated", "logs", "transcript.jsonl")
					if info, err := os.Stat(tPath); err == nil && !seen[tPath] {
						seen[tPath] = true
						all = append(all, Found{Path: tPath, Changed: info.ModTime()})
					}
				}
			}
		}
		_ = f.Close()
	}

	// 2. Direct scan of brain/ directory as fallback
	brainDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain")
	if entries, err := os.ReadDir(brainDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || !antigravityUUIDRegex.MatchString(e.Name()) {
				continue
			}
			tPath := filepath.Join(brainDir, e.Name(), ".system_generated", "logs", "transcript.jsonl")
			if seen[tPath] {
				continue
			}
			if info, err := os.Stat(tPath); err == nil {
				seen[tPath] = true
				all = append(all, Found{Path: tPath, Changed: info.ModTime()})
			}
		}
	}

	slices.SortFunc(all, func(a, b Found) int {
		return b.Changed.Compare(a.Changed)
	})
	return all
}

// AntigravityConversationID extracts the conversation UUID from an Antigravity transcript path.
func AntigravityConversationID(path string) string {
	if m := antigravityUUIDRegex.FindStringSubmatch(path); len(m) > 1 {
		return m[1]
	}
	return ""
}

// NewAntigravityReader creates a reader following an Antigravity transcript.jsonl.
func NewAntigravityReader(path string) *Reader {
	return newReader(path, &antigravityParser{})
}

type antigravityParser struct{}

func (*antigravityParser) fresh() parser { return &antigravityParser{} }

type antigravityToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type antigravityRecord struct {
	StepIndex       int                   `json:"step_index"`
	Source          string                `json:"source"`
	Type            string                `json:"type"`
	Status          string                `json:"status"`
	CreatedAt       time.Time             `json:"created_at"`
	Content         string                `json:"content"`
	InputTokens     int64                 `json:"input_tokens"`
	OutputTokens    int64                 `json:"output_tokens"`
	CacheReadTokens int64                 `json:"cache_read_tokens"`
	ToolCalls       []antigravityToolCall `json:"tool_calls"`
}

func (p *antigravityParser) parse(line []byte, r *Reader) {
	var rec antigravityRecord
	if json.Unmarshal(line, &rec) != nil {
		return
	}

	if rec.InputTokens > 0 || rec.OutputTokens > 0 {
		r.Context = rec.InputTokens + rec.OutputTokens
	}

	switch rec.Type {
	case "USER_INPUT":
		content := extractUserContent(rec.Content)
		if content != "" {
			r.add(proto.TranscriptEntry{
				Kind: proto.TranscriptUser,
				Time: rec.CreatedAt,
				Text: content,
			})
		}
	case "PLANNER_RESPONSE":
		if rec.Content != "" {
			r.add(proto.TranscriptEntry{
				Kind: proto.TranscriptAssistant,
				Time: rec.CreatedAt,
				Text: rec.Content,
			})
		}
		for _, tc := range rec.ToolCalls {
			r.add(proto.TranscriptEntry{
				Kind:   proto.TranscriptTool,
				Time:   rec.CreatedAt,
				Text:   tc.Name,
				Detail: string(tc.Args),
			})
		}
	}
}

func extractUserContent(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, userReqOpen)
	end := strings.Index(raw, userReqClose)
	if start >= 0 && end > start {
		return strings.TrimSpace(raw[start+len(userReqOpen) : end])
	}
	return raw
}
