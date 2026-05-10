package template

import (
	"strings"
	"testing"
	"time"
)

// parityCtx is shared across keyword parity tests so we can drive both
// RenderMessage (lenient) and RenderMessageStrict with the same inputs and
// compare their output lines.
func parityCtx() Context {
	return Context{
		LocalIP:      "127.0.0.10",
		ServerIP:     "127.0.0.20",
		Users:        7,
		UserID:       3,
		ClockTick:    1200,
		DynamicID:    42,
		LastMessage:  "INVITE sip:alice@example.com SIP/2.0\r\nTo: <sip:bob@example.com>\r\n\r\n",
		LastHeaders:  map[string][]string{"To": {"<sip:bob@example.com>"}},
		CallNumber:   1,
		MessageIndex: 2,
	}
}

func TestRenderServerIPLenientAndStrict(t *testing.T) {
	t.Parallel()

	ctx := parityCtx()
	raw := "X-Server-IP: [server_ip]\r\n\r\n"

	gotLenient := RenderMessage(raw, ctx)
	gotStrict, err := RenderMessageStrict(raw, ctx)
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("server_ip parity mismatch:\n lenient=%q\n strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-Server-IP: 127.0.0.20") {
		t.Fatalf("expected server_ip from ServerIP, got %q", gotLenient)
	}

	fallback := ctx
	fallback.ServerIP = ""
	gotFallback := RenderMessage(raw, fallback)
	if !strings.Contains(gotFallback, "X-Server-IP: 127.0.0.10") {
		t.Fatalf("expected server_ip to fall back to LocalIP, got %q", gotFallback)
	}
}

func TestRenderUsersLenientAndStrict(t *testing.T) {
	t.Parallel()

	ctx := parityCtx()
	raw := "X-Users: [users]\r\n\r\n"

	gotLenient := RenderMessage(raw, ctx)
	gotStrict, err := RenderMessageStrict(raw, ctx)
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("users parity mismatch: lenient=%q strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-Users: 7") {
		t.Fatalf("expected users helper, got %q", gotLenient)
	}
}

func TestRenderUserIDLenientAndStrict(t *testing.T) {
	t.Parallel()

	ctx := parityCtx()
	raw := "X-UserID: [userid]\r\n\r\n"

	gotLenient := RenderMessage(raw, ctx)
	gotStrict, err := RenderMessageStrict(raw, ctx)
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("userid parity mismatch: lenient=%q strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-UserID: 3") {
		t.Fatalf("expected userid helper, got %q", gotLenient)
	}
}

func TestRenderLastRequestURILenientAndStrict(t *testing.T) {
	t.Parallel()

	ctx := parityCtx()
	raw := "X-URI: [last_Request_URI]\r\n\r\n"

	gotLenient := RenderMessage(raw, ctx)
	gotStrict, err := RenderMessageStrict(raw, ctx)
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("last_Request_URI parity mismatch: lenient=%q strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-URI: sip:alice@example.com") {
		t.Fatalf("expected last_Request_URI rendered from LastMessage start line, got %q", gotLenient)
	}
}

func TestRenderDateUsesGMT(t *testing.T) {
	t.Parallel()

	raw := "Date: [date]\r\n\r\n"

	gotLenient := RenderMessage(raw, Context{})
	gotStrict, err := RenderMessageStrict(raw, Context{})
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	for _, got := range []string{gotLenient, gotStrict} {
		if !strings.Contains(got, " GMT") {
			t.Fatalf("expected RFC2822 Date with GMT zone, got %q", got)
		}
		if strings.Contains(got, " UTC") {
			t.Fatalf("Date must not render with UTC zone, got %q", got)
		}
		// Sanity: the rendered Date must be parseable by Go's RFC1123 parser
		// because RFC1123 uses GMT in the canonical form too.
		dateLine := strings.TrimPrefix(strings.SplitN(got, "\r\n", 2)[0], "Date: ")
		if _, perr := time.Parse(time.RFC1123, dateLine); perr != nil {
			t.Fatalf("rendered Date %q does not parse as RFC1123: %v", dateLine, perr)
		}
	}
}

func TestRenderTDMMapStubLenientAndStrict(t *testing.T) {
	t.Parallel()

	raw := "X-TDM: [tdmmap]\r\n\r\n"

	gotLenient := RenderMessage(raw, Context{})
	gotStrict, err := RenderMessageStrict(raw, Context{})
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("tdmmap parity mismatch: lenient=%q strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-TDM: 0.0.0/0") {
		t.Fatalf("expected tdmmap stub 0.0.0/0, got %q", gotLenient)
	}
}

func TestRenderClockTickAndDynamicIDLenientAndStrict(t *testing.T) {
	t.Parallel()

	ctx := parityCtx()
	raw := "X-Tick: [clock_tick+2]\r\nX-Dynamic: [dynamic_id]\r\n\r\n"

	gotLenient := RenderMessage(raw, ctx)
	gotStrict, err := RenderMessageStrict(raw, ctx)
	if err != nil {
		t.Fatalf("RenderMessageStrict error: %v", err)
	}
	if gotLenient != gotStrict {
		t.Fatalf("clock_tick/dynamic_id parity mismatch: lenient=%q strict=%q", gotLenient, gotStrict)
	}
	if !strings.Contains(gotLenient, "X-Tick: 1202") {
		t.Fatalf("expected clock_tick with offset, got %q", gotLenient)
	}
	if !strings.Contains(gotLenient, "X-Dynamic: 42") {
		t.Fatalf("expected dynamic_id passthrough, got %q", gotLenient)
	}
}
