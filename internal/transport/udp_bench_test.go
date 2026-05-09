package transport

import (
	"net"
	"runtime"
	"strconv"
	"testing"
)

// BenchmarkSharedUDP_NewClose measures bind/teardown cost with parallel reuseport listeners.
func BenchmarkSharedUDP_NewClose(b *testing.B) {
	n := 8
	if runtime.NumCPU() < n {
		n = runtime.NumCPU()
	}
	if n < 1 {
		n = 1
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s, err := NewSharedUDPWithReceivers("127.0.0.1:0", n)
		if err != nil {
			b.Fatal(err)
		}
		_ = s.Close()
	}
}

// BenchmarkSharedUDP_RoundTrip sends UDP to self through SharedUDP (listen → readLoop → channel).
func BenchmarkSharedUDP_RoundTrip(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	const payloadLen = 512
	payload := make([]byte, payloadLen)

	rc := runtime.NumCPU()
	if rc > maxUDPReceivers {
		rc = maxUDPReceivers
	}
	if rc < 1 {
		rc = 1
	}

	s, err := NewSharedUDPWithReceivers("127.0.0.1:0", rc)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	la := net.JoinHostPort("127.0.0.1", strconv.Itoa(s.LocalPort()))
	remote, err := net.ResolveUDPAddr("udp", la)
	if err != nil {
		b.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	b.SetBytes(payloadLen)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := conn.Write(payload); err != nil {
			b.Fatal(err)
		}
		p := <-s.Receive()
		if len(p.Data) != payloadLen {
			b.Fatalf("got len %d want %d", len(p.Data), payloadLen)
		}
	}
}
