package ui

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/enowx/enowx/core/hosts"
	"github.com/enowx/enowx/ui/theme"
)

// Hosts › Edit… has "Connect at start", on for a host already set up; turned off it
// is saved with the host, and the dialog keeps its switch in its frame however
// narrow the window.
func TestEditHostConnectAtStart(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ENOWX_HOME", home)
	store, err := hosts.Open(filepath.Join(home, "hosts.json"))
	if err != nil {
		t.Fatal(err)
	}
	h, err := store.Add(hosts.Host{Name: "vps-sg-1", Address: "root@127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	a := New(Options{Hosts: store})
	posted := make(chan func(), 64)
	a.onMain = func(fn func()) { posted <- fn }
	a.openEditHost(h)
	if !a.addHost.startConnect {
		t.Fatal("a host already set up isn't set to connect at start")
	}
	for _, width := range []int{900, 520} {
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(theme.Window())
			ui.Column(c).Width(float32(width)).Height(900).Background(theme.Background).Children(func() { a.addHostDialog(c) })
		}, width, 900)
		tt.Frame()
		tt.Frame()
		r, ok := tt.Find("Connect at start")
		if !ok {
			t.Fatalf("%d: no Connect at start in %q", width, tt.Texts())
		}
		if r.X < 0 || r.X+r.W > float32(width) {
			t.Fatalf("%d: the switch is out of the window: %+v", width, r)
		}
		if width == 520 {
			// The switch sits at the row's right edge, centered on its words.
			for x := r.X + r.W; x < float32(width) && a.addHost.startConnect; x += 12 {
				tt.ClickAt(x, r.Y+10)
				tt.Frame()
			}
		}
	}
	if a.addHost.startConnect {
		t.Fatal("the switch didn't turn off")
	}
	a.saveHost()
	saved, _ := store.Get(h.ID)
	if !saved.NoStartConnect {
		t.Fatal("Connect at start off wasn't saved with the host")
	}
	again, err := hosts.Open(filepath.Join(home, "hosts.json"))
	if err != nil {
		t.Fatal(err)
	}
	if kept, _ := again.Get(h.ID); !kept.NoStartConnect {
		t.Fatal("not kept in the host list's file")
	}
}
