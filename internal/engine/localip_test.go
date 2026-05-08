package engine

import (
	"net"
	"testing"
)

func TestResolveLocalIPUsesExplicitAddress(t *testing.T) {
	t.Parallel()
	got := resolveLocalIP(0, "10.11.12.13", "8.8.8.8", 53)
	if got != "10.11.12.13" {
		t.Fatalf("got %q want 10.11.12.13", got)
	}
}

func TestResolveLocalIPWildcardDiscoversTowardPeer(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	defer pc.Close()
	addr := pc.LocalAddr().(*net.UDPAddr)
	got := resolveLocalIP(0, "0.0.0.0", addr.IP.String(), addr.Port)
	if got != "127.0.0.1" {
		t.Fatalf("got %q want 127.0.0.1", got)
	}
}

func TestIsWildcardLocalIP(t *testing.T) {
	t.Parallel()
	for _, ip := range []string{"", "0.0.0.0", "::"} {
		if !isWildcardLocalIP(ip) {
			t.Fatalf("expected wildcard for %q", ip)
		}
	}
	if isWildcardLocalIP("127.0.0.1") {
		t.Fatal("127.0.0.1 should not be wildcard")
	}
}
