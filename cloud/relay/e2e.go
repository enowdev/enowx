package relay

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// End to end between a member and the host they visit, so the relay passes
// bytes it can't read. The host has a key of its own (X25519), whose public
// half it announces in its team's space, which the cloud keeps sealed and
// can't forge. The member's app opens a channel to that key alone:
//
//  1. the member sends a fresh key e;
//  2. the host answers with a fresh key e2 and a proof that it holds the
//     announced key (a box only DH(e, host key) opens);
//  3. both derive a key each way from DH(e, host key) and DH(e, e2).
//
// The relay sees two fresh public keys and a proof; without the host's key
// it can neither read the records that follow nor stand in for the host,
// and the fresh keys mean a key stolen later can't read what was recorded.
// Who the member is, the host learns inside the channel, from their ticket.

const (
	magic      = "ENXR1"
	recordMax  = 16 << 10 // plaintext of a record
	handshakeT = 15 * time.Second
)

// ErrHostKey is a host that couldn't prove it holds the key its team
// announced: not the host, or a relay standing in for it.
var ErrHostKey = errors.New("relay: the host didn't prove it holds the key its team announced")

// Key is a host's own key.
type Key struct{ priv *ecdh.PrivateKey }

// LoadKey reads the host's key from file, making one the first time.
func LoadKey(file string) (*Key, error) {
	if data, err := os.ReadFile(file); err == nil {
		priv, err := ecdh.X25519().NewPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("relay: the host key in %s: %w", file, err)
		}
		return &Key{priv}, nil
	}
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(file, priv.Bytes(), 0o600); err != nil {
		return nil, fmt.Errorf("relay: write the host key: %w", err)
	}
	return &Key{priv}, nil
}

// NewKey is a key kept in memory alone (tests).
func NewKey() *Key {
	priv, _ := ecdh.X25519().GenerateKey(rand.Reader)
	return &Key{priv}
}

// Public is the key's public half, as the host announces it.
func (k *Key) Public() string { return base64.StdEncoding.EncodeToString(k.priv.PublicKey().Bytes()) }

func transcript(parts ...[]byte) []byte {
	h := sha256.New()
	h.Write([]byte("enowx-relay-e2e-v1"))
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

func derive(secret, salt []byte, info string, n int) ([]byte, error) {
	return hkdf.Key(sha256.New, secret, salt, info, n)
}

// Client opens the channel to the host whose announced public key is
// hostKey, over nc (the member's end).
func Client(nc net.Conn, hostKey string) (net.Conn, error) {
	raw, err := base64.StdEncoding.DecodeString(hostKey)
	if err != nil {
		return nil, fmt.Errorf("relay: the host's announced key: %w", err)
	}
	S, err := ecdh.X25519().NewPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("relay: the host's announced key: %w", err)
	}
	e, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	_ = nc.SetDeadline(time.Now().Add(handshakeT))
	defer nc.SetDeadline(time.Time{})
	ePub := e.PublicKey().Bytes()
	if _, err := nc.Write(append([]byte(magic), ePub...)); err != nil {
		return nil, err
	}
	var answer [32 + 2 + chacha20poly1305.Overhead]byte
	if _, err := io.ReadFull(nc, answer[:]); err != nil {
		return nil, fmt.Errorf("relay: the host's answer: %w", err)
	}
	e2Pub := answer[:32]
	dh1, err := e.ECDH(S)
	if err != nil {
		return nil, ErrHostKey
	}
	h := transcript(raw, ePub)
	h2 := transcript(h, e2Pub)
	k1, err := derive(dh1, h, "host proof", chacha20poly1305.KeySize)
	if err != nil {
		return nil, err
	}
	box, _ := chacha20poly1305.New(k1)
	if ok, err := box.Open(nil, make([]byte, chacha20poly1305.NonceSize), answer[32:], h2); err != nil || string(ok) != "ok" {
		return nil, ErrHostKey
	}
	E2, err := ecdh.X25519().NewPublicKey(e2Pub)
	if err != nil {
		return nil, ErrHostKey
	}
	dh2, err := e.ECDH(E2)
	if err != nil {
		return nil, ErrHostKey
	}
	return newSecure(nc, dh1, dh2, h2, true)
}

// Server answers a member's channel over nc (the host's end).
func (k *Key) Server(nc net.Conn) (net.Conn, error) {
	_ = nc.SetDeadline(time.Now().Add(handshakeT))
	defer nc.SetDeadline(time.Time{})
	var hello [len(magic) + 32]byte
	if _, err := io.ReadFull(nc, hello[:]); err != nil {
		return nil, fmt.Errorf("relay: the member's hello: %w", err)
	}
	if string(hello[:len(magic)]) != magic {
		return nil, errors.New("relay: not a member's channel")
	}
	ePub := hello[len(magic):]
	E, err := ecdh.X25519().NewPublicKey(ePub)
	if err != nil {
		return nil, err
	}
	dh1, err := k.priv.ECDH(E)
	if err != nil {
		return nil, err
	}
	e2, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	dh2, err := e2.ECDH(E)
	if err != nil {
		return nil, err
	}
	e2Pub := e2.PublicKey().Bytes()
	h := transcript(k.priv.PublicKey().Bytes(), ePub)
	h2 := transcript(h, e2Pub)
	k1, err := derive(dh1, h, "host proof", chacha20poly1305.KeySize)
	if err != nil {
		return nil, err
	}
	box, _ := chacha20poly1305.New(k1)
	proof := box.Seal(nil, make([]byte, chacha20poly1305.NonceSize), []byte("ok"), h2)
	if _, err := nc.Write(append(e2Pub, proof...)); err != nil {
		return nil, err
	}
	return newSecure(nc, dh1, dh2, h2, false)
}

// secure is a channel's records: a length, then a box sealed with the
// direction's key and a counter, so a record dropped, repeated or moved by
// the relay fails to open and ends the channel.
type secure struct {
	net.Conn
	rmu      sync.Mutex // the reading side: in, rbuf, rn
	in       cipher.AEAD
	rn       uint64
	rbuf     []byte
	wmu      sync.Mutex // the writing side: out, wn
	out      cipher.AEAD
	wn       uint64
	readErr  error
	writeErr error
}

func newSecure(nc net.Conn, dh1, dh2, salt []byte, client bool) (net.Conn, error) {
	keys, err := derive(append(append([]byte{}, dh1...), dh2...), salt, "enowx-relay keys", 2*chacha20poly1305.KeySize)
	if err != nil {
		return nil, err
	}
	toHost, _ := chacha20poly1305.New(keys[:32])
	toMember, _ := chacha20poly1305.New(keys[32:])
	s := &secure{Conn: nc, in: toMember, out: toHost}
	if !client {
		s.in, s.out = toHost, toMember
	}
	return s, nil
}

func nonce(n uint64) []byte {
	var b [chacha20poly1305.NonceSize]byte
	binary.LittleEndian.PutUint64(b[4:], n)
	return b[:]
}

func (s *secure) Read(p []byte) (int, error) {
	s.rmu.Lock()
	defer s.rmu.Unlock()
	for len(s.rbuf) == 0 {
		if s.readErr != nil {
			return 0, s.readErr
		}
		var hdr [2]byte
		if _, err := io.ReadFull(s.Conn, hdr[:]); err != nil {
			s.readErr = err
			return 0, err
		}
		n := int(binary.BigEndian.Uint16(hdr[:]))
		if n < chacha20poly1305.Overhead || n > recordMax+chacha20poly1305.Overhead {
			s.readErr = errors.New("relay: a record of a wrong size")
			return 0, s.readErr
		}
		rec := make([]byte, n)
		if _, err := io.ReadFull(s.Conn, rec); err != nil {
			s.readErr = err
			return 0, err
		}
		plain, err := s.in.Open(rec[:0], nonce(s.rn), rec, nil)
		if err != nil {
			s.readErr = errors.New("relay: a record that doesn't open: changed on its way")
			return 0, s.readErr
		}
		s.rn++
		s.rbuf = plain
	}
	n := copy(p, s.rbuf)
	s.rbuf = s.rbuf[n:]
	return n, nil
}

func (s *secure) Write(p []byte) (int, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	done := 0
	for done < len(p) {
		chunk := p[done:min(len(p), done+recordMax)]
		rec := make([]byte, 2, 2+len(chunk)+chacha20poly1305.Overhead)
		rec = s.out.Seal(rec, nonce(s.wn), chunk, nil)
		binary.BigEndian.PutUint16(rec, uint16(len(rec)-2))
		if _, err := s.Conn.Write(rec); err != nil {
			s.writeErr = err
			return done, err
		}
		s.wn++
		done += len(chunk)
	}
	return done, nil
}
