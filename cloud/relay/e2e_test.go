package relay

import (
	"errors"
	"io"
	"net"
	"testing"
)

// A member's channel opens to the host holding the announced key, carries
// bytes both ways, and refuses one that doesn't hold it or a record changed
// on its way.
func TestChannel(t *testing.T) {
	host := NewKey()
	a, b := net.Pipe()
	got := make(chan net.Conn, 1)
	go func() {
		sc, err := host.Server(b)
		if err != nil {
			t.Error(err)
		}
		got <- sc
	}()
	mc, err := Client(a, host.Public())
	if err != nil {
		t.Fatal(err)
	}
	hc := <-got
	go mc.Write(make([]byte, 40000)) // past one record
	buf := make([]byte, 40000)
	if _, err := io.ReadFull(hc, buf); err != nil {
		t.Fatal(err)
	}
	go hc.Write([]byte("back"))
	small := make([]byte, 4)
	if _, err := io.ReadFull(mc, small); err != nil || string(small) != "back" {
		t.Fatalf("%q %v", small, err)
	}

	a, b = net.Pipe()
	go NewKey().Server(b) // another host than the one announced
	if _, err := Client(a, host.Public()); !errors.Is(err, ErrHostKey) {
		t.Fatalf("a host without the announced key: %v", err)
	}
}

func TestChangedRecordFails(t *testing.T) {
	host := NewKey()
	a, b := net.Pipe()
	mid1, mid2 := net.Pipe() // what passes from member to host, through here
	go func() {
		sc, _ := host.Server(mid2)
		buf := make([]byte, 8)
		if _, err := sc.Read(buf); err == nil {
			t.Error("a changed record opened")
		}
	}()
	go func() { // the handshake both ways, then a record with a bit flipped
		hello := make([]byte, 37)
		io.ReadFull(b, hello)
		mid1.Write(hello)
		answer := make([]byte, 50)
		io.ReadFull(mid1, answer)
		b.Write(answer)
		rec := make([]byte, 2+5+16)
		io.ReadFull(b, rec)
		rec[4] ^= 1
		mid1.Write(rec)
	}()
	mc, err := Client(a, host.Public())
	if err != nil {
		t.Fatal(err)
	}
	mc.Write([]byte("hello"))
}
