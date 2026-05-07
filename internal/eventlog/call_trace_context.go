package eventlog

import (
	"context"
	"crypto/sha256"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// emitContext returns a context whose span context carries a synthetic trace
// derived from SIP Call-ID when attrs contain a non-empty "call_id" string.
// The mapping is stable so all OTLP log records for the same call share one
// trace_id (OpenTelemetry Logs trace correlation).
//
// When there is no call_id, the background context is returned and the SDK
// leaves trace_id empty as before.
func emitContext(attrs map[string]any) context.Context {
	callID := callIDFromAttrs(attrs)
	if callID == "" {
		return context.Background()
	}
	sc := syntheticSpanContext(callID)
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func callIDFromAttrs(attrs map[string]any) string {
	if attrs == nil {
		return ""
	}
	v, ok := attrs["call_id"]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s
}

func syntheticSpanContext(callID string) trace.SpanContext {
	sum := sha256.Sum256([]byte(callID))
	var tid trace.TraceID
	copy(tid[:], sum[:16])
	var sid trace.SpanID
	copy(sid[:], sum[16:24])
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
}
