package agent

import (
	"os"
	"strings"
	"testing"
)

// openCodeOnWindows is GOD of Agents on OpenCode 1.18 in a conversation, idle, as its screen
// read on Windows (issue #51): the composer's rows are bars alone, then its commands' hint.
func openCodeOnWindows(draft string) string {
	row := "  ┃"
	if draft != "" {
		row = "  ┃  " + draft
	}
	return "  Done: the worker's branch is merged.\n\n" +
		"     ▣  Goa · wz/deepseek-v4.1-flash · 2m 10s\n\n" +
		"  ┃\n" + row + "\n" +
		"  ┃                                    ctrl+p commands • OpenCode 1.18.35\n"
}

// OpenCode's composer with nothing typed reads as empty, so lines for GOD of Agents go in;
// with a draft it doesn't, nor while OpenCode works or asks to allow something.
func TestOpenCodeComposer(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	cases := []struct {
		name   string
		screen string
		empty  bool
	}{
		{"Linux, its placeholder", read("opencode-1.18-linux-empty.txt"), true},
		{"Linux, a draft", read("opencode-1.18-linux-draft.txt"), false},
		{"Linux, an update's dialog over it", read("opencode-1.18-linux-dialog.txt"), false},
		{"Windows, in a conversation", openCodeOnWindows(""), true},
		{"Windows, a draft", openCodeOnWindows("sekalian cek routes-nya"), false},
		{"Windows, a draft that starts as the placeholder", openCodeOnWindows("Ask anything about the routes"), false},
		{"at work", strings.Replace(openCodeOnWindows(""), "ctrl+p commands", "esc interrupt", 1), false},
		{"asking to allow something", openCodeOnWindows("") + "Permission required: bash rm -rf dist\n", false},
	}
	for _, c := range cases {
		if got := PromptEmpty(c.screen); got != c.empty {
			t.Errorf("%s: PromptEmpty = %v, want %v", c.name, got, c.empty)
		}
	}
}

// A bar alone, or a box of them that isn't OpenCode's, isn't an empty composer.
func TestABarAloneIsNoComposer(t *testing.T) {
	for _, screen := range []string{
		"┃\n┃\n┃\n",
		"some output\n  ┃\n  ┃\n  ┃ total 3 files\n",
		"│\n│ ❯ draft\n│\n",
	} {
		if PromptEmpty(screen) {
			t.Errorf("read as an empty composer:\n%s", screen)
		}
	}
}
