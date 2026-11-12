package ui

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/ui/theme"
)

type fakeWindow struct{ did []string }

func (w *fakeWindow) Close()            { w.did = append(w.did, "close") }
func (w *fakeWindow) Minimize()         { w.did = append(w.did, "minimize") }
func (w *fakeWindow) ToggleMaximize()   { w.did = append(w.did, "maximize") }
func (w *fakeWindow) ToggleFullScreen() { w.did = append(w.did, "full screen") }

// The title bar's own buttons close, minimize, and take the window full
// screen (macOS) or maximize it.
func TestWindowButtons(t *testing.T) {
	a := New(Options{})
	w := &fakeWindow{}
	a.winCtl = w
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(900).Height(60).Background(theme.Ground).Children(func() { a.titleBar(c) })
	}, 900, 60)
	tt.Frame()
	for _, b := range []string{"Zoom", "Minimize", "Close"} {
		if err := tt.Click(b); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	shot(t, tt, "title-bar.png")
	zoom := "maximize"
	if zoomGoesFullScreen {
		zoom = "full screen"
	}
	if got := w.did; len(got) != 3 || got[0] != zoom || got[1] != "minimize" || got[2] != "close" {
		t.Fatalf("did %q", got)
	}
}

func TestWindowsStyleButtons(t *testing.T) {
	a := New(Options{})
	a.state.WindowControlStyle = WindowControlStyleWindows
	w := &fakeWindow{}
	a.winCtl = w
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(900).Height(60).Background(theme.Ground).Children(func() { a.titleBar(c) })
	}, 900, 60)
	tt.Frame()

	// Mac dots Zoom should not be present
	if _, ok := tt.Find("Zoom"); ok {
		t.Fatal("mac style Zoom button should not exist in Windows style")
	}

	// Click Windows buttons: Maximize, Minimize, Close
	for _, b := range []string{"Maximize", "Minimize", "Close"} {
		if err := tt.Click(b); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	zoom := "maximize"
	if zoomGoesFullScreen {
		zoom = "full screen"
	}
	if got := w.did; len(got) != 3 || got[0] != zoom || got[1] != "minimize" || got[2] != "close" {
		t.Fatalf("did %q", got)
	}
}

func TestWindowControlStyleAppearanceSetting(t *testing.T) {
	file := filepath.Join(t.TempDir(), "state.json")
	app := New(Options{StateFile: file})
	app.host = LocalHost()

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(1000).Height(800).Background(theme.Ground).Children(func() { app.appearancePage(c) })
	}, 1000, 800)
	tt.Frame()

	// Initially mac style
	if app.isWindowsControlStyle() {
		t.Fatal("expected default to be mac style")
	}

	// Click "Windows style"
	if err := tt.Click("Windows style"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	if !app.isWindowsControlStyle() {
		t.Fatal("expected Windows style to be active")
	}

	app.saveState()
	reopened := New(Options{StateFile: file})
	if reopened.state.WindowControlStyle != WindowControlStyleWindows {
		t.Fatalf("expected reopened to have windows control style, got %q", reopened.state.WindowControlStyle)
	}

	// Switch back to "Mac style"
	if err := tt.Click("Mac style"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if app.isWindowsControlStyle() {
		t.Fatal("expected Mac style to be active")
	}
}

