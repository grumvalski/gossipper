package engine

import (
	"strings"
	"testing"
)

func TestNormalizeSIPScenarioLineIndentFlushStartLineRestIndented(t *testing.T) {
	t.Parallel()
	raw := "REGISTER sip:example.com SIP/2.0\r\n" +
		"      Via: SIP/2.0/TCP 10.0.0.1:5060;branch=z9hG4bK-1\r\n" +
		"      Content-Length: 0\r\n\r\n"
	got := normalizeSIPScenarioLineIndent(raw)
	want := "REGISTER sip:example.com SIP/2.0\r\n" +
		"Via: SIP/2.0/TCP 10.0.0.1:5060;branch=z9hG4bK-1\r\n" +
		"Content-Length: 0\r\n\r\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeSIPScenarioLineIndentStripsCommonXMLPadding(t *testing.T) {
	t.Parallel()
	raw := "      REGISTER sip:example.com SIP/2.0\r\n" +
		"      Via: SIP/2.0/TCP 10.0.0.1:5060;branch=z9hG4bK-1\r\n" +
		"      Content-Length: 0\r\n\r\n"
	got := normalizeSIPScenarioLineIndent(raw)
	want := "REGISTER sip:example.com SIP/2.0\r\n" +
		"Via: SIP/2.0/TCP 10.0.0.1:5060;branch=z9hG4bK-1\r\n" +
		"Content-Length: 0\r\n\r\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeSIPScenarioLineIndentAlsoDedentsBody(t *testing.T) {
	t.Parallel()
	raw := "    INVITE sip:a SIP/2.0\r\n" +
		"    Content-Type: application/sdp\r\n" +
		"    Content-Length: 3\r\n" +
		"\r\n" +
		"    v=0\r\n"
	got := normalizeSIPScenarioLineIndent(raw)
	if !strings.HasSuffix(got, "\r\n\r\nv=0\r\n") {
		t.Fatalf("body should be dedented, got %q", got)
	}
	if !strings.HasPrefix(got, "INVITE sip:a SIP/2.0\r\n") {
		t.Fatalf("headers should be dedented, got %q", got)
	}
}
