package recovery

import (
	"regexp"
)

var (
	// Patterns for sensitive secrets: API keys, tokens, auth headers.
	apiKeyRegex    = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|password|bearer|auth|gemini|openai|anthropic)[\s:=]+["']?([a-zA-Z0-9_\-\.]{8,})["']?`)
	jwtRegex       = regexp.MustCompile(`ey[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}\.[a-zA-Z0-9_-]{10,}`)
	skKeyRegex     = regexp.MustCompile(`(sk-[a-zA-Z0-9]{20,}|AIzaSy[a-zA-Z0-9_-]{33}|ghp_[a-zA-Z0-9]{36})`)
	credValueRegex = regexp.MustCompile(`(?i)(bearer\s+)[a-zA-Z0-9_\-\.]+`)
)

// SanitizeText removes API keys, credentials, and tokens from a string.
func SanitizeText(text string) string {
	if text == "" {
		return ""
	}
	s := apiKeyRegex.ReplaceAllString(text, `$1=[REDACTED]`)
	s = jwtRegex.ReplaceAllString(s, `[REDACTED_JWT]`)
	s = skKeyRegex.ReplaceAllString(s, `[REDACTED_KEY]`)
	s = credValueRegex.ReplaceAllString(s, `${1}[REDACTED_TOKEN]`)
	return s
}

// SanitizeFailureContext returns a sanitized copy of FailureContext.
func SanitizeFailureContext(fc *FailureContext) *FailureContext {
	if fc == nil {
		return nil
	}
	return &FailureContext{
		ErrorCode:    fc.ErrorCode,
		ErrorMessage: SanitizeText(fc.ErrorMessage),
		StackTrace:   SanitizeText(fc.StackTrace),
		Recoverable:  fc.Recoverable,
	}
}
