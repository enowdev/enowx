// Package relay connects a team's members to a shared host through the
// cloud's relay, end to end encrypted (e2e.go): Host is the host's side, a
// listener of members' connections; Dial the member's.
package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/enowx/enowx/cloud/api"
)

const readLimit = 1 << 20

// Dial reaches the host whose announced key is hostKey through the relay at
// relayURL (the ticket's), handing the relay ticket so it pairs the two.
// The connection it gives is the channel to the host.
func Dial(ctx context.Context, relayURL, ticket, hostKey string) (net.Conn, error) {
	ws, resp, err := websocket.Dial(ctx, relayURL, &websocket.DialOptions{HTTPHeader: http.Header{api.HeaderTicket: {ticket}}})
	if err != nil {
		return nil, refusal(resp, err)
	}
	ws.SetReadLimit(readLimit)
	nc := websocket.NetConn(context.Background(), ws, websocket.MessageBinary)
	sc, err := Client(nc, hostKey)
	if err != nil {
		nc.Close()
		return nil, err
	}
	return sc, nil
}

// refusal is why the relay said no, in its words when it gave them.
func refusal(resp *http.Response, err error) error {
	if resp == nil || resp.Body == nil {
		return err
	}
	var e api.Error
	if json.NewDecoder(resp.Body).Decode(&e) == nil && e.Error != "" {
		return &Refused{Status: resp.StatusCode, Message: e.Error}
	}
	return fmt.Errorf("relay: %s", resp.Status)
}

// Refused is the relay saying no.
type Refused struct {
	Status  int
	Message string
}

func (e *Refused) Error() string { return "relay: " + e.Message }

// Host keeps the host's connection to the relay at URL (the cloud's
// address) with Token, the host's own, and gives members' channels, opened
// to Key, as a listener's connections.
type Host struct {
	URL, Token string
	Key        *Key
	// Retry is the first wait before connecting again; it doubles to
	// RetryMax. Zero: a second, and five minutes.
	Retry, RetryMax time.Duration
	// Log notes what goes wrong; nil, nothing.
	Log func(msg string, args ...any)

	mu       sync.Mutex // guards ws, err, up
	ws       *websocket.Conn
	err      string
	up       bool
	conns    chan net.Conn
	ctx      context.Context
	stop     context.CancelFunc
	stopOnce sync.Once
}

// Start connects and keeps connecting until Close.
func (h *Host) Start() {
	h.conns = make(chan net.Conn)
	h.ctx, h.stop = context.WithCancel(context.Background())
	go h.run()
}

// Status says whether the host is connected to the relay, and why not.
func (h *Host) Status() (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.up, h.err
}

func (h *Host) wsURL(path string) string {
	base := strings.TrimSuffix(h.URL, "/")
	base = strings.Replace(strings.Replace(base, "https://", "wss://", 1), "http://", "ws://", 1)
	return base + path
}

func (h *Host) auth() *websocket.DialOptions {
	return &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + h.Token}}}
}

func (h *Host) run() {
	wait := h.Retry
	if wait <= 0 {
		wait = time.Second
	}
	most := h.RetryMax
	if most <= 0 {
		most = 5 * time.Minute
	}
	first := wait
	for h.ctx.Err() == nil {
		began := time.Now()
		err := h.serve()
		if h.ctx.Err() != nil {
			return
		}
		h.mu.Lock()
		h.up, h.err = false, err.Error()
		h.mu.Unlock()
		if h.Log != nil {
			h.Log("relay: connect again", "err", err, "in", wait)
		}
		if time.Since(began) > time.Minute {
			wait = first // it stayed up a while: the drop is news, not a pattern
		}
		select {
		case <-h.ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, most)
	}
}

// serve holds one connection to the relay, answering its calls.
func (h *Host) serve() error {
	ctx, cancel := context.WithTimeout(h.ctx, 20*time.Second)
	ws, resp, err := websocket.Dial(ctx, h.wsURL(api.PathRelayHost), h.auth())
	cancel()
	if err != nil {
		return refusal(resp, err)
	}
	h.mu.Lock()
	h.ws, h.up, h.err = ws, true, ""
	h.mu.Unlock()
	defer ws.CloseNow()
	for {
		_, data, err := ws.Read(h.ctx)
		if err != nil {
			return fmt.Errorf("relay: the connection dropped: %w", err)
		}
		var call api.RelayCall
		if json.Unmarshal(data, &call) != nil || call.ID == "" {
			continue
		}
		go h.answer(call)
	}
}

// answer opens the call's connection and the member's channel over it.
func (h *Host) answer(call api.RelayCall) {
	ctx, cancel := context.WithTimeout(h.ctx, 15*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, h.wsURL(strings.Replace(api.PathRelayAccept, "{id}", call.ID, 1)), h.auth())
	if err != nil {
		return
	}
	ws.SetReadLimit(readLimit)
	nc := websocket.NetConn(h.ctx, ws, websocket.MessageBinary)
	sc, err := h.Key.Server(nc)
	if err != nil {
		nc.Close()
		return
	}
	select {
	case h.conns <- &member{Conn: sc, from: call.From}:
	case <-ctx.Done():
		sc.Close()
	}
}

// member is a member's channel, from the address the relay saw.
type member struct {
	net.Conn
	from string
}

func (m *member) RemoteAddr() net.Addr { return Addr(m.from) }

// Addr is where the relay saw a member come from.
type Addr string

func (a Addr) Network() string { return "relay" }
func (a Addr) String() string  { return string(a) }

// Accept waits for a member's channel.
func (h *Host) Accept() (net.Conn, error) {
	select {
	case c := <-h.conns:
		return c, nil
	case <-h.ctx.Done():
		return nil, net.ErrClosed
	}
}

// Close stops connecting, and drops the connection to the relay.
func (h *Host) Close() error {
	h.stopOnce.Do(func() {
		h.stop()
		h.mu.Lock()
		if h.ws != nil {
			h.ws.CloseNow()
		}
		h.up = false
		h.mu.Unlock()
	})
	return nil
}

// Addr is the relay's, as a listener's.
func (h *Host) Addr() net.Addr { return Addr("relay") }

var _ net.Listener = (*Host)(nil)
