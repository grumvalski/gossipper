package transport

import "testing"

func TestSharedUDPOpenClose(t *testing.T) {
	s, err := NewSharedUDP("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
