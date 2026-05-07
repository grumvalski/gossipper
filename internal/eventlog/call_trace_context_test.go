package eventlog

import (
	"crypto/sha256"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestCallIDFromAttrs(t *testing.T) {
	if got := callIDFromAttrs(nil); got != "" {
		t.Fatalf("nil attrs: got %q", got)
	}
	if got := callIDFromAttrs(map[string]any{}); got != "" {
		t.Fatalf("empty attrs: got %q", got)
	}
	if got := callIDFromAttrs(map[string]any{"call_id": ""}); got != "" {
		t.Fatalf("empty string: got %q", got)
	}
	if got := callIDFromAttrs(map[string]any{"call_id": "  "}); got != "" {
		t.Fatalf("whitespace: got %q", got)
	}
	if got := callIDFromAttrs(map[string]any{"call_id": 42}); got != "" {
		t.Fatalf("non-string: got %q", got)
	}
	if got := callIDFromAttrs(map[string]any{"call_id": "x"}); got != "x" {
		t.Fatalf("string: got %q want x", got)
	}
}

func TestSyntheticSpanContextStable(t *testing.T) {
	a := syntheticSpanContext("bench-call-id-0001@127.0.0.1")
	b := syntheticSpanContext("bench-call-id-0001@127.0.0.1")
	if a.TraceID() != b.TraceID() || a.SpanID() != b.SpanID() {
		t.Fatalf("expected identical span contexts for same Call-ID")
	}
	c := syntheticSpanContext("other-call")
	if a.TraceID() == c.TraceID() {
		t.Fatalf("expected different trace ids for different Call-IDs")
	}
}

func TestEmitContextBackgroundWithoutCallID(t *testing.T) {
	ctx := emitContext(map[string]any{"sip.method": "INVITE"})
	if trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatalf("expected invalid span context without call_id")
	}
}

func TestEmitContextCarriesTraceForCallID(t *testing.T) {
	ctx := emitContext(map[string]any{"call_id": "abc"})
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		t.Fatal("expected valid span context")
	}
	var want trace.TraceID
	sum := sha256.Sum256([]byte("abc"))
	copy(want[:], sum[:16])
	if sc.TraceID() != want {
		t.Fatalf("TraceID: got %s want %s", sc.TraceID(), want)
	}
	var wantSpan trace.SpanID
	copy(wantSpan[:], sum[16:24])
	if sc.SpanID() != wantSpan {
		t.Fatalf("SpanID: got %s want %s", sc.SpanID(), wantSpan)
	}
}
