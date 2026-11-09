package agent

import (
	"strings"
	"testing"
)

// claudeOnConPTY is the end of Claude Code's screen in a session's terminal on Windows, as
// the terminal's text reads it. ConPTY writes a rule as wide as the screen with no line
// break after it, so the rule and the next row read as one soft-wrapped line: the composer
// comes right after the rule above it, and the status line after the rule below.
func claudeOnConPTY(draft string) string {
	rule := strings.Repeat("─", 98)
	return "● Homeserver sudah tersambung.\n\n" +
		"✻ Worked for 23s · done 1:10 PM\n\n" +
		rule + "❯\u00a0" + draft + "\n" +
		strings.Repeat("─", 77) + "  \u00a0~\\.enowx-app\\goa\u00a0\ue0b0\u00a0⎇\u00a0no\u00a0git\n" +
		"Opus\u00a05.5\u00a0\ue0b0\u00a0Ctx\u00a0Used:\u00a013.0%\u00a0\ue0b0\u00a05h:\u00a00.0%\n" +
		"  ⏵⏵ bypass permissions on (shift+tab to cycle)\n"
}

// A composer read joined to the rule above it is still a composer: empty when nothing
// follows its mark, and holding a draft when something does. Lines for GOD of Agents are
// typed in only into an empty one, so with the join read as a draft they never went.
func TestPromptEmptyUnderARuleWrappedOntoIt(t *testing.T) {
	if !PromptEmpty(claudeOnConPTY("")) {
		t.Error("an empty composer joined to its rule reads as holding a draft")
	}
	if PromptEmpty(claudeOnConPTY("sekalian kita update")) {
		t.Error("a draft joined to its rule reads as an empty composer")
	}
	if !TurnOverShown(claudeOnConPTY("")) {
		t.Error("the turn's end isn't seen")
	}
}

// A rule's run is cut from what follows it, and nothing else is: a line that only starts
// with a dash or two, as a turn's time drawn between rules, stays whole.
func TestSplitWrappedRule(t *testing.T) {
	rule := strings.Repeat("─", 40)
	cases := []struct {
		line string
		want []string
	}{
		{rule + "❯", []string{rule, "❯"}},
		{rule + "  ~\\goa", []string{rule, "~\\goa"}},
		{rule, []string{rule}},
		{"─ Worked for 1m 03s ─", []string{"─ Worked for 1m 03s ─"}},
		{"──── note", []string{"──── note"}},
		{"❯ hello", []string{"❯ hello"}},
	}
	for _, c := range cases {
		if got := splitWrappedRule(c.line); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("splitWrappedRule(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}
