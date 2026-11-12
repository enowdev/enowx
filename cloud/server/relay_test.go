package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/enowx/enowx/cloud/api"
	relaypkg "github.com/enowx/enowx/cloud/relay"
	"github.com/enowx/enowx/cloud/store"
	"github.com/enowx/enowx/cloud/store/storetest"
)

type relayTeam struct {
	srv             *httptest.Server
	api             *Server
	db              *store.DB
	team            string
	adaHost, adaTok string
	bobTok          string
	key             *relaypkg.Key
	host            *relaypkg.Host
}

// startRelayTeam is a cloud with Ada's host on its relay, echoing what a
// member sends, and Bob a member of her team.
func startRelayTeam(t *testing.T, cfg Config) *relayTeam {
	ctx := context.Background()
	cfg.URL = "http://cloud.test" // the relay's address comes from the request in tests
	db := storetest.Open(t)
	cs := New(cfg, db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(cs)
	t.Cleanup(srv.Close)
	device := func(gh int64, login string) (string, string) {
		acc, _ := db.GitHubAccount(ctx, gh, login, login, "", "")
		d, tok, _ := db.AddDevice(ctx, acc.ID, login+"'s", api.DeviceHost, make([]byte, 32))
		return d.ID, tok
	}
	r := &relayTeam{srv: srv, api: cs, db: db, key: relaypkg.NewKey()}
	r.adaHost, r.adaTok = device(1, "ada")
	_, r.bobTok = device(2, "bob")
	name := "Design"
	var team api.Team
	send(t, "POST", srv.URL+api.PathTeams, r.adaTok, api.TeamParams{Name: &name}, &team)
	var inv api.Invite
	send(t, "POST", srv.URL+"/v1/teams/"+team.ID+"/invites", r.adaTok, api.InviteParams{Role: api.RoleMember}, &inv)
	send(t, "POST", srv.URL+api.PathJoin, r.bobTok, api.JoinParams{Token: inv.Token}, nil)
	r.team = team.ID
	r.host = &relaypkg.Host{URL: srv.URL, Token: r.adaTok, Key: r.key, Retry: 50 * time.Millisecond}
	r.host.Start()
	t.Cleanup(func() { r.host.Close() })
	go func() {
		for {
			c, err := r.host.Accept()
			if err != nil {
				return
			}
			go io.Copy(c, c)
		}
	}()
	r.waitUp(t)
	return r
}

func (r *relayTeam) waitUp(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if up, _ := r.host.Status(); up {
			return
		}
		if time.Now().After(deadline) {
			_, why := r.host.Status()
			t.Fatalf("the host never reached the relay: %s", why)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (r *relayTeam) ticket(t *testing.T) string {
	t.Helper()
	var tk api.HostTicket
	if code, msg := send(t, "POST", r.srv.URL+"/v1/teams/"+r.team+"/tickets", r.bobTok, api.HostTicketParams{Host: r.adaHost}, &tk); code != http.StatusOK {
		t.Fatalf("a ticket: %d %s", code, msg)
	}
	if tk.Relay == "" {
		t.Fatal("a ticket without the relay's address")
	}
	return tk.Ticket
}

func (r *relayTeam) dial(ticket string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return relaypkg.Dial(ctx, "ws"+r.srv.URL[len("http"):]+api.PathRelayConnect, ticket, r.key.Public())
}

// A member with a ticket is paired with the host; what passes between them
// is sealed, so the relay can't read it; a ticket pairs once.
func TestRelayPairsAndCantRead(t *testing.T) {
	r := startRelayTeam(t, Config{})
	var mu sync.Mutex
	var seen bytes.Buffer
	relayTap = func(b []byte) { mu.Lock(); seen.Write(b); mu.Unlock() }
	t.Cleanup(func() { relayTap = nil })

	tk := r.ticket(t)
	c, err := r.dial(tk)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	secret := []byte("cat ~/projects/launch-plan.md: the terminal's screen")
	c.Write(secret)
	back := make([]byte, len(secret))
	if _, err := io.ReadFull(c, back); err != nil || !bytes.Equal(back, secret) {
		t.Fatalf("echoed through the relay: %q %v", back, err)
	}
	mu.Lock()
	n := seen.Len()
	leaked := bytes.Contains(seen.Bytes(), []byte("launch-plan")) || bytes.Contains(seen.Bytes(), []byte("terminal"))
	mu.Unlock()
	if n < 2*len(secret) {
		t.Fatalf("the relay passed %d bytes, less than the visit", n)
	}
	if leaked {
		t.Fatal("the relay read what a member and the host said")
	}
	if _, err := r.dial(tk); !refusedWith(err, http.StatusForbidden) {
		t.Fatalf("a ticket paired twice: %v", err)
	}
}

// A forged ticket, or one of someone no longer in the team, pairs nobody.
func TestRelayRefusesTickets(t *testing.T) {
	r := startRelayTeam(t, Config{})
	if _, err := r.dial("enxt_forged"); !refusedWith(err, http.StatusForbidden) {
		t.Fatalf("a forged ticket: %v", err)
	}
	tk := r.ticket(t)
	ctx := context.Background()
	ada, _ := r.db.GitHubAccount(ctx, 1, "ada", "ada", "", "")
	bob, _ := r.db.GitHubAccount(ctx, 2, "bob", "bob", "", "")
	if err := r.db.RemoveMember(ctx, r.team, ada.ID, bob.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.dial(tk); !refusedWith(err, http.StatusForbidden) {
		t.Fatalf("one no longer in the team was paired: %v", err)
	}
}

// A team has so many connections at once through the relay.
func TestRelayTeamLimit(t *testing.T) {
	r := startRelayTeam(t, Config{Relay: RelayLimits{Pairs: 1}})
	c, err := r.dial(r.ticket(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.dial(r.ticket(t)); !refusedWith(err, http.StatusTooManyRequests) {
		t.Fatalf("past the team's connections: %v", err)
	}
	c.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		c, err := r.dial(r.ticket(t))
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("a connection closed left its room taken: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A host whose connection to the relay drops connects again, and members
// reach it as before.
func TestRelayHostReconnects(t *testing.T) {
	r := startRelayTeam(t, Config{})
	r.api.relay.mu.Lock()
	h := r.api.relay.hosts[r.adaHost]
	r.api.relay.mu.Unlock()
	h.ws.CloseNow() // as a proxy or the network would
	deadline := time.Now().Add(3 * time.Second)
	for up, _ := r.host.Status(); up; up, _ = r.host.Status() {
		if time.Now().After(deadline) {
			t.Fatal("the host didn't notice its connection dropped")
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.waitUp(t)
	deadline = time.Now().Add(5 * time.Second)
	for {
		c, err := r.dial(r.ticket(t))
		if err == nil {
			c.Write([]byte("hi"))
			b := make([]byte, 2)
			if _, err := io.ReadFull(c, b); err != nil {
				t.Fatal(err)
			}
			c.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("after the host's connection dropped: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func refusedWith(err error, status int) bool {
	var r *relaypkg.Refused
	return errors.As(err, &r) && r.Status == status
}

// idleCutter stands for a proxy on the way that cuts a connection which
// carried nothing either way for cut, as Cloudflare's does: it passes
// connections to target.
func idleCutter(t *testing.T, target string, cut time.Duration) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			in, err := l.Accept()
			if err != nil {
				return
			}
			out, err := net.Dial("tcp", target)
			if err != nil {
				in.Close()
				continue
			}
			var mu sync.Mutex
			last := time.Now()
			pass := func(dst, src net.Conn) {
				buf := make([]byte, 32<<10)
				for {
					n, err := src.Read(buf)
					if n > 0 {
						mu.Lock()
						last = time.Now()
						mu.Unlock()
						dst.Write(buf[:n])
					}
					if err != nil {
						in.Close()
						out.Close()
						return
					}
				}
			}
			go pass(out, in)
			go pass(in, out)
			go func() {
				for range time.Tick(cut / 10) {
					mu.Lock()
					idle := time.Since(last)
					mu.Unlock()
					if idle > cut {
						in.Close()
						out.Close()
						return
					}
				}
			}()
		}
	}()
	return l.Addr().String()
}

// A pair that carries nothing for longer than a proxy on the way lets a
// connection idle stays up: the relay pings both its sides.
func TestRelayQuietPairStaysUp(t *testing.T) {
	r := startRelayTeam(t, Config{Relay: RelayLimits{Ping: 100 * time.Millisecond}})
	cut := "http://" + idleCutter(t, r.srv.Listener.Addr().String(), 500*time.Millisecond)
	// Ada's host reaches the relay through the proxy too.
	r.host.Close()
	host := &relaypkg.Host{URL: cut, Token: r.adaTok, Key: r.key, Retry: 50 * time.Millisecond}
	host.Start()
	t.Cleanup(func() { host.Close() })
	go func() {
		for {
			c, err := host.Accept()
			if err != nil {
				return
			}
			go io.Copy(c, c)
		}
	}()
	for up, _ := host.Status(); !up; up, _ = host.Status() {
		time.Sleep(10 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := relaypkg.Dial(ctx, "ws"+cut[len("http"):]+api.PathRelayConnect, r.ticket(t), r.key.Public())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// A member's client reads all along, as the app's does.
	got := make(chan []byte, 1)
	go func() {
		back := make([]byte, 10)
		n, _ := io.ReadFull(c, back)
		got <- back[:n]
	}()
	time.Sleep(2 * time.Second) // four times what the proxy lets idle
	msg := []byte("still here")
	if _, err := c.Write(msg); err != nil {
		t.Fatalf("the quiet pair was cut: %v", err)
	}
	select {
	case back := <-got:
		if !bytes.Equal(back, msg) {
			t.Fatalf("the quiet pair was cut: %q", back)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing came back")
	}
}
