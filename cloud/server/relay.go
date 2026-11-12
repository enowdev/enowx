package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
		"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/enowx/enowx/cloud/api"
	"github.com/enowx/enowx/cloud/store"
)

// The relay (api.PathRelayHost): hosts sharing projects keep a connection
// here, members connect with a ticket, and the relay pairs them and passes
// bytes it can't read (they are end to end encrypted, cloud/relay). It
// checks the ticket without using it (the host redeems it, as over any
// other path), lets a ticket pair once, and holds each team to a number of
// connections and a rate of bytes. State is this server's memory: one
// relay per cloud.

// RelayLimits are a team's share of the relay; zero takes the default.
type RelayLimits struct {
	// Pairs is how many members' connections a team has at once.
	Pairs int
	// BytesPerSecond is a team's rate, both ways together, all its
	// connections together; Burst is how much may go at once above it.
	BytesPerSecond, Burst int
}

const (
	relayPairs     = 32
	relayRate      = 4 << 20 // bytes a second
	relayBurst     = 16 << 20
	relayAnswer    = 15 * time.Second // for the host to open a call
	relayPing      = 30 * time.Second
	relayConnects  = 30 // a minute, from one address
	relayReadLimit = 1 << 20
)

// relay is the hosts connected and the calls waiting; mu guards all of it.
type relay struct {
	mu      sync.Mutex
	hosts   map[string]*relayHost // by device
	calls   map[string]*relayCall // by ID
	tickets map[string]time.Time  // hashes of tickets paired, until they expire
	teams   map[string]*teamShare
}

type relayHost struct {
	ws  *websocket.Conn
	wmu sync.Mutex // writes to ws
}

type relayCall struct {
	host  string
	conn  chan *websocket.Conn
	done  chan struct{}
	start time.Time
}

// teamShare is what a team uses of the relay now.
type teamShare struct {
	pairs  int
	tokens float64
	last   time.Time
}

func (s *Server) relayLimits() RelayLimits {
	l := s.cfg.Relay
	if l.Pairs <= 0 {
		l.Pairs = relayPairs
	}
	if l.BytesPerSecond <= 0 {
		l.BytesPerSecond = relayRate
	}
	if l.Burst <= 0 {
		l.Burst = relayBurst
	}
	return l
}

// relayURL is where members connect, "" when the server has no address.
func (s *Server) relayURL() string {
	base := strings.TrimSuffix(s.cfg.URL, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		return "wss://" + strings.TrimPrefix(base, "https://") + api.PathRelayConnect
	case strings.HasPrefix(base, "http://"):
		return "ws://" + strings.TrimPrefix(base, "http://") + api.PathRelayConnect
	}
	return ""
}

func randomID() string {
	var b [18]byte
	rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// relayHostConn is a host's standing connection: calls go out on it.
func (s *Server) relayHostConn(w http.ResponseWriter, r *http.Request) {
	dev := asker(r).Device.ID
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	h := &relayHost{ws: ws}
	s.relay.mu.Lock()
	if s.relay.hosts == nil {
		s.relay.hosts = map[string]*relayHost{}
	}
	old := s.relay.hosts[dev]
	s.relay.hosts[dev] = h
	s.relay.mu.Unlock()
	if old != nil {
		old.ws.Close(websocket.StatusPolicyViolation, "this host connected again")
	}
	s.log.Info("relay host", "device", dev, "state", "connected")
	defer func() {
		s.relay.mu.Lock()
		if s.relay.hosts[dev] == h {
			delete(s.relay.hosts, dev)
		}
		s.relay.mu.Unlock()
		ws.CloseNow()
		s.log.Info("relay host", "device", dev, "state", "gone")
	}()
	ctx := ws.CloseRead(r.Context()) // the host sends nothing on it
	t := time.NewTicker(relayPing)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			h.wmu.Lock()
			err := ws.Ping(pctx)
			h.wmu.Unlock()
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// relayAccept is the host's connection for one call.
func (s *Server) relayAccept(w http.ResponseWriter, r *http.Request) {
	s.relay.mu.Lock()
	call := s.relay.calls[r.PathValue("id")]
	s.relay.mu.Unlock()
	if call == nil || call.host != asker(r).Device.ID {
		s.fail(w, http.StatusNotFound, "no such call for this host")
		return
	}
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	select {
	case call.conn <- ws:
		<-call.done // the member's side passes the bytes, then lets go
	default:
		ws.Close(websocket.StatusPolicyViolation, "this call is answered")
	}
}

// relayConnect is a member's connection, with their ticket: paired with
// the host's for the ticket's host.
func (s *Server) relayConnect(w http.ResponseWriter, r *http.Request) {
	from := clientAddr(r)
	if !s.limits.allow("relay", from, relayConnects) {
		w.Header().Set("Retry-After", "60")
		s.fail(w, http.StatusTooManyRequests, "too many tries from here: wait a minute")
		return
	}
	ticket := r.Header.Get(api.HeaderTicket)
	team, account, host, err := s.db.PeekTicket(r.Context(), ticket)
	if errors.Is(err, store.ErrTicket) || ticket == "" {
		s.fail(w, http.StatusForbidden, "the ticket is no good: used, expired, or of one no longer in the team")
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	limits := s.relayLimits()
	s.relay.mu.Lock()
	sum := sha256.Sum256([]byte(ticket))
	hash := string(sum[:])
	now := time.Now()
	for k, exp := range s.relay.tickets {
		if now.After(exp) {
			delete(s.relay.tickets, k)
		}
	}
	if _, seen := s.relay.tickets[hash]; seen {
		s.relay.mu.Unlock()
		s.fail(w, http.StatusForbidden, "this ticket was used on the relay already: ask for another")
		return
	}
	hc := s.relay.hosts[host]
	if hc == nil {
		s.relay.mu.Unlock()
		s.fail(w, http.StatusServiceUnavailable, "the host isn't connected to the relay: it may be off, or not sharing")
		return
	}
	if s.relay.teams == nil {
		s.relay.teams = map[string]*teamShare{}
		s.relay.calls = map[string]*relayCall{}
		s.relay.tickets = map[string]time.Time{}
	}
	ts := s.relay.teams[team]
	if ts == nil {
		ts = &teamShare{tokens: float64(limits.Burst), last: now}
		s.relay.teams[team] = ts
	}
	if ts.pairs >= limits.Pairs {
		s.relay.mu.Unlock()
		s.fail(w, http.StatusTooManyRequests, "your team has as many connections through the relay as it may: close one")
		return
	}
	ts.pairs++
	s.relay.tickets[hash] = now.Add(api.TicketLife)
	call := &relayCall{host: host, conn: make(chan *websocket.Conn, 1), done: make(chan struct{}), start: now}
	id := randomID()
	s.relay.calls[id] = call
	s.relay.mu.Unlock()
	defer func() {
		s.relay.mu.Lock()
		delete(s.relay.calls, id)
		ts.pairs--
		s.relay.mu.Unlock()
		close(call.done)
	}()

	msg, _ := json.Marshal(api.RelayCall{ID: id, From: from})
	wctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	hc.wmu.Lock()
	err = hc.ws.Write(wctx, websocket.MessageText, msg)
	hc.wmu.Unlock()
	cancel()
	if err != nil {
		s.fail(w, http.StatusServiceUnavailable, "the host isn't connected to the relay")
		return
	}
	var hostWS *websocket.Conn
	select {
	case hostWS = <-call.conn:
	case <-time.After(relayAnswer):
		s.fail(w, http.StatusGatewayTimeout, "the host didn't answer")
		return
	case <-r.Context().Done():
		return
	}
	defer hostWS.CloseNow()
	memberWS, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer memberWS.CloseNow()
	memberWS.SetReadLimit(relayReadLimit)
	hostWS.SetReadLimit(relayReadLimit)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	m := websocket.NetConn(ctx, memberWS, websocket.MessageBinary)
	h := websocket.NetConn(ctx, hostWS, websocket.MessageBinary)
	var up, down int64
	done := make(chan struct{}, 2)
	go func() { up = s.relayCopy(h, m, ts); stop(); done <- struct{}{} }()
	go func() { down = s.relayCopy(m, h, ts); stop(); done <- struct{}{} }()
	<-done
	m.Close()
	h.Close()
	<-done
	// Who reached whom and how much: all the relay knows of a visit.
	s.log.Info("relay visit", "team", team, "account", account, "host", host, "up", up, "down", down, "s", int(time.Since(call.start).Seconds()))
}

// relayTap sees what the relay passes, for tests.
var relayTap func([]byte)

// relayCopy passes bytes from src to dst within the team's rate.
func (s *Server) relayCopy(dst io.Writer, src io.Reader, ts *teamShare) int64 {
	buf := make([]byte, 32<<10)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			s.relayWait(ts, n)
			if relayTap != nil {
				relayTap(buf[:n])
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total
			}
			total += int64(n)
		}
		if err != nil {
			return total
		}
	}
}

// relayWait takes n bytes from the team's allowance, waiting for it.
func (s *Server) relayWait(ts *teamShare, n int) {
	l := s.relayLimits()
	n = min(n, l.Burst)
	for {
		s.relay.mu.Lock()
		now := time.Now()
		ts.tokens = min(float64(l.Burst), ts.tokens+now.Sub(ts.last).Seconds()*float64(l.BytesPerSecond))
		ts.last = now
		if ts.tokens >= float64(n) {
			ts.tokens -= float64(n)
			s.relay.mu.Unlock()
			return
		}
		need := (float64(n) - ts.tokens) / float64(l.BytesPerSecond)
		s.relay.mu.Unlock()
		time.Sleep(time.Duration(need*float64(time.Second)) + time.Millisecond)
	}
}
