package agent

import (
	"regexp"
	"strings"
)

// OpenCode's composer is a box whose rows start with a heavy bar, and has no prompt mark:
//
//	┃
//	┃  Ask anything… "Fix a TODO in the codebase"    (the placeholder, on its first screen)
//	┃
//	┃  Build · Big Pickle OpenCode Zen               (its agent and model)
//	╹▀▀▀▀▀▀▀▀
//
// In a conversation the placeholder isn't shown, and on Windows the last row may read as
// its commands' hint. Only that box counts: a bar alone isn't a composer for other agents.

// openCodePlaceholder is the composer's placeholder, OpenCode's exact words.
var openCodePlaceholder = regexp.MustCompile(`^┃\s*Ask anything(…|\.\.\.)(\s+"[^"]*")?$`)

// openCodeDialog is a dialog's title over the screen, its close key at the right: what is
// typed then goes to the dialog (an update's, a model's pick), not the composer.
var openCodeDialog = regexp.MustCompile(`\s{2,}esc$`)

// openCodeChrome is the box's last row: the agent and model it runs, or its commands' hint.
var openCodeChrome = regexp.MustCompile(`^┃\s+(\S.* · |.*ctrl\+p commands)`)

// openCodeEmpty says lines end on OpenCode's composer with nothing typed in it: the last
// box of bar rows, its last row OpenCode's, every other row bare or the placeholder, and
// nothing at work nor asking to allow something. A row with anything else is a draft.
func openCodeEmpty(lines []string) bool {
	end := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "┃") {
			end = i
			break
		}
	}
	start := end
	for start > 0 && strings.HasPrefix(lines[start-1], "┃") {
		start--
	}
	if end < 0 || end-start < 2 || !openCodeChrome.MatchString(lines[end]) {
		return false
	}
	if lastMatch(lines, openCodeDialog) >= 0 {
		return false
	}
	for _, row := range lines[start:end] {
		if row != "┃" && !openCodePlaceholder.MatchString(row) {
			return false
		}
	}
	// Under the box are its hints and a tip, whose words change: only a prompt to allow
	// something counts there.
	return !workingShown.MatchString(strings.Join(lines, "\n")) && !permissionPrompts.MatchString(strings.Join(lines[end+1:], "\n"))
}
