package daemon

import (
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/enowx/enowx/cloud/relay"
)

// The ways members reach a shared host (docs/cloud.md, Hosts and sessions):
// the cloud's relay, which needs no address of the host's, and the host's
// Tailscale address when it has one (members of an older enowX know only
// that). Both feed one listener, the team listener, whose checks are the
// same whichever way a member came.

// relayRetry is the first wait before the host connects to the relay again.
var relayRetry = time.Second

// relayKeyFile is the host's key members' channels open to (cloud/relay).
func (s *Server) relayKeyFile() string { return filepath.Join(s.dataDir, "team-relay.key") }

// startRelay connects the host to its cloud's relay, signed in: nil
// without a cloud or a sign-in.
func (s *Server) startRelay() (*relay.Host, string, error) {
	st, err := s.cloud.store.Get()
	if err != nil || !st.SignedIn() || st.URL == "" {
		return nil, "", nil
	}
	key, err := relay.LoadKey(s.relayKeyFile())
	if err != nil {
		return nil, "", err
	}
	h := &relay.Host{URL: st.URL, Token: st.Token, Key: key, Retry: relayRetry}
	if s.Log != nil {
		h.Log = s.Log.Warn
	}
	h.Start()
	return h, key.Public(), nil
}

// restartSharing opens the team listener again, as the cloud the host
// signs in to changed.
func (s *Server) restartSharing() {
	s.stopSharing()
	s.startSharing()
}

// fanIn is a listener of several: it gives what any of them accepts, and
// closes them all. Its mutex guards closed.
type fanIn struct {
	conns  chan net.Conn
	done   chan struct{}
	mu     sync.Mutex
	closed bool
	srcs   []net.Listener
}

func newFanIn() *fanIn { return &fanIn{conns: make(chan net.Conn), done: make(chan struct{})} }

// add takes l's connections in, until it closes.
func (f *fanIn) add(l net.Listener) {
	f.srcs = append(f.srcs, l)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			select {
			case f.conns <- c:
			case <-f.done:
				c.Close()
				return
			}
		}
	}()
}

func (f *fanIn) Accept() (net.Conn, error) {
	select {
	case c := <-f.conns:
		return c, nil
	case <-f.done:
		return nil, net.ErrClosed
	}
}

func (f *fanIn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.done)
		for _, l := range f.srcs {
			l.Close()
		}
	}
	return nil
}

func (f *fanIn) Addr() net.Addr { return relay.Addr("team") }
