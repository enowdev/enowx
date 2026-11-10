//go:build !windows

package ui

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/enowx/enowx/client"
	"github.com/enowx/enowx/core/hosts"
	"github.com/enowx/enowx/daemon"
	"github.com/enowx/enowx/proto"
)

// At the app's start every saved host connects in the background, at once and each on
// its own, and its sessions list as soon as it is up: one that doesn't answer holds no
// other and counts as offline when its time is up, one that needs the user waits for
// them, and one set not to connect at start waits until the user opens it.
func TestHostsConnectAtStart(t *testing.T) {
	srv, err := daemon.NewServer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	x, y := net.Pipe()
	go srv.ServeConn(x)
	maker, err := client.New(y, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer maker.Close()
	s, err := maker.CreateSession(proto.SessionCreateParams{Command: []string{"sh"}, Dir: t.TempDir(), Cols: 80, Rows: 10})
	if err != nil {
		t.Fatal(err)
	}

	most := watchConnectMost
	watchConnectMost = 700 * time.Millisecond
	t.Cleanup(func() { watchConnectMost = most })
	a, run := watchedApp(t)
	hang := make(chan struct{})
	defer close(hang)
	var mu sync.Mutex
	dialed := map[string]int{}
	a.watchDial = func(h hosts.Host) (*client.Client, error) {
		mu.Lock()
		dialed[h.ID]++
		mu.Unlock()
		switch h.ID {
		case "down":
			<-hang
			return nil, errors.New("gone")
		case "needs-you":
			return nil, errNeedsYou
		}
		time.Sleep(400 * time.Millisecond) // a server across the world
		p, q := net.Pipe()
		go srv.ServeConn(p)
		return client.New(q, nil)
	}
	a.hostList = []hosts.Host{
		{ID: "vps-a", Name: "vps-a"}, {ID: "vps-b", Name: "vps-b"}, {ID: "vps-c", Name: "vps-c"},
		{ID: "down", Name: "down"}, {ID: "needs-you", Name: "needs-you"},
		{ID: "later", Name: "later", NoStartConnect: true},
	}
	a.host = LocalHost()
	began := time.Now()
	a.syncWatches()
	defer a.stopWatches()

	online := func(id string) bool { w := a.watches[id]; return w != nil && w.status.state == hostOnline }
	run(func() bool { return online("vps-a") && online("vps-b") && online("vps-c") }, "the hosts online", 10*time.Second)
	if took := time.Since(began); took > 1200*time.Millisecond {
		t.Fatalf("three hosts of 400 ms each took %s: not connected at once", took)
	}
	listed := func() bool {
		for _, info := range a.watches["vps-b"].status.sessions {
			if info.ID == s.ID {
				return true
			}
		}
		return false
	}
	run(listed, "a host's sessions listed once it is up", 5*time.Second)
	run(func() bool { return a.watches["down"].status.state == hostOffline }, "a host that doesn't answer offline", 5*time.Second)
	if got := hostSummary(a.watches["down"].status); got != "offline · retrying" {
		t.Fatalf("a host down reads %q", got)
	}
	run(func() bool { return a.watches["needs-you"].status.state == hostNeedsYou }, "a host that needs the user", 5*time.Second)
	if _, ok := a.watches["later"]; ok {
		t.Fatal("a host set not to connect at start was connected to")
	}

	// Back in sight, or awake: a host offline tries again now, not when its wait ends;
	// one that needs the user doesn't, nothing having changed for it.
	mu.Lock()
	downBefore, needsBefore := dialed["down"], dialed["needs-you"]
	mu.Unlock()
	a.pokeWatches()
	tried := func() bool { mu.Lock(); defer mu.Unlock(); return dialed["down"] > downBefore }
	run(tried, "the host down tried again at once", 3*time.Second)
	mu.Lock()
	if dialed["needs-you"] != needsBefore {
		t.Fatal("a host that needs the user was tried again by itself")
	}
	mu.Unlock()

	// Opened once, it is watched from then on like the others.
	a.opened = map[string]bool{"later": true}
	a.syncWatches()
	run(func() bool { return online("later") }, "a host the user opened watched", 5*time.Second)
}
