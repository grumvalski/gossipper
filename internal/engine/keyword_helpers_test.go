package engine

import (
	"math"
	"testing"
	"time"

	"github.com/sipcapture/gossipper/internal/sip"
)

func TestEngineNextDynamicIDIncrementsAndWraps(t *testing.T) {
	t.Parallel()

	e := New(Config{})
	if got := e.nextDynamicID(); got != 1 {
		t.Fatalf("first dynamic_id = %d, want 1", got)
	}
	if got := e.nextDynamicID(); got != 2 {
		t.Fatalf("second dynamic_id = %d, want 2", got)
	}

	e.dynamicID.Store(math.MaxInt32 - 1)
	if got := e.nextDynamicID(); got != math.MaxInt32 {
		t.Fatalf("dynamic_id at boundary = %d, want %d", got, int64(math.MaxInt32))
	}
	if got := e.nextDynamicID(); got != 0 {
		t.Fatalf("dynamic_id wrap = %d, want 0", got)
	}
	if got := e.nextDynamicID(); got != 1 {
		t.Fatalf("post-wrap dynamic_id = %d, want 1", got)
	}
}

func TestEngineClockTickReflectsUptime(t *testing.T) {
	t.Parallel()

	e := New(Config{})

	first := e.clockTick()
	if first < 0 {
		t.Fatalf("clock_tick must be non-negative, got %d", first)
	}
	time.Sleep(15 * time.Millisecond)
	second := e.clockTick()
	if second <= first {
		t.Fatalf("clock_tick must be monotonically non-decreasing across sleep, got first=%d second=%d", first, second)
	}
}

func TestEngineMailboxRegistryNormalizesCallID(t *testing.T) {
	t.Parallel()

	registry := newMailboxRegistry(nil)

	// Register with bare Call-ID; messages bearing prefix///id form must still match.
	bare := "dialog-99"
	prefixed := "leg-A///" + bare
	ch := registry.register(bare)
	defer registry.unregister(bare)

	msg := sip.GetMessage()
	defer sip.PutMessage(msg)
	msg.Headers = map[string][]string{"Call-ID": {prefixed}}
	msg.Raw = "INVITE sip:peer@example SIP/2.0\r\nCall-ID: " + prefixed + "\r\n\r\n"

	registry.dispatchMessagePtr(msg)
	select {
	case got := <-ch:
		if got != msg {
			t.Fatalf("mailbox received different message")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for normalized Call-ID dispatch")
	}

	// Reverse direction: register prefixed form, lookup with bare normalized form.
	bare2 := "dialog-100"
	prefixed2 := "abc///" + bare2
	ch2 := registry.register(prefixed2)
	defer registry.unregister(prefixed2)

	msg2 := sip.GetMessage()
	defer sip.PutMessage(msg2)
	msg2.Headers = map[string][]string{"Call-ID": {bare2}}
	msg2.Raw = "INVITE sip:peer@example SIP/2.0\r\nCall-ID: " + bare2 + "\r\n\r\n"

	registry.dispatchMessagePtr(msg2)
	select {
	case got := <-ch2:
		if got != msg2 {
			t.Fatalf("mailbox received different message")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for bare Call-ID dispatch into prefixed registration")
	}
}

func TestEngineCommandCallIDStripsTripleSlashPrefix(t *testing.T) {
	t.Parallel()

	raw := "INVITE sip:peer@example SIP/2.0\r\nCall-ID: corr-leader///cmd-77\r\nFrom: alice\r\n\r\n"
	if got := commandCallID(raw, ""); got != "cmd-77" {
		t.Fatalf("commandCallID = %q, want %q", got, "cmd-77")
	}

	rawBare := "INVITE sip:peer@example SIP/2.0\r\nCall-ID: cmd-78\r\nFrom: alice\r\n\r\n"
	if got := commandCallID(rawBare, ""); got != "cmd-78" {
		t.Fatalf("commandCallID bare = %q, want %q", got, "cmd-78")
	}
}
