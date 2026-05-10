package engine

import (
	"strings"
	"testing"

	"github.com/sipcapture/gossipper/internal/scenario"
	templ "github.com/sipcapture/gossipper/internal/template"
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

// TestParsedSendDedentsIndentedCDATA verifies the full parser → engine dedent
// pipeline for <send>: an XML-indented CDATA block must produce a flush-left
// SIP message after normalizeSIPScenarioLineIndent.
func TestParsedSendDedentsIndentedCDATA(t *testing.T) {
	t.Parallel()
	sc, err := scenario.ParseString(`<?xml version="1.0" encoding="UTF-8"?>
<scenario name="indented-send">
  <send>
    <![CDATA[
      INVITE sip:a SIP/2.0
      Call-ID: abc
      Content-Type: application/sdp
      Content-Length: 3

      v=0
    ]]>
  </send>
</scenario>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}
	if len(sc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sc.Commands))
	}
	got := normalizeSIPScenarioLineIndent(sc.Commands[0].SendText)
	if !strings.HasPrefix(got, "INVITE sip:a SIP/2.0\r\n") {
		t.Fatalf("headers should start flush-left, got %q", got)
	}
	if !strings.Contains(got, "\r\n\r\nv=0") {
		t.Fatalf("body should be flush-left after CRLF separator, got %q", got)
	}
	if strings.Contains(got, "\r\n  ") || strings.Contains(got, "\r\n\t") {
		t.Fatalf("no rendered line should keep leading indent, got %q", got)
	}
}

// TestParsedSendCmdDedentsIndentedCDATA verifies that 3PCC <sendCmd> payloads
// also have their CDATA indentation stripped (engine.go and init.go run the
// same normalize step before RenderMessageStrict).
func TestParsedSendCmdDedentsIndentedCDATA(t *testing.T) {
	t.Parallel()
	sc, err := scenario.ParseString(`<?xml version="1.0" encoding="UTF-8"?>
<scenario name="indented-sendcmd">
  <sendCmd dest="s1"><![CDATA[
    Call-ID: [call_id]
    X-Value: hello-cmd
  ]]></sendCmd>
</scenario>`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}
	if len(sc.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sc.Commands))
	}
	got := normalizeSIPScenarioLineIndent(sc.Commands[0].SendText)
	if !strings.HasPrefix(got, "Call-ID: [call_id]\r\n") {
		t.Fatalf("first header should be flush-left, got %q", got)
	}
	if !strings.Contains(got, "\r\nX-Value: hello-cmd") {
		t.Fatalf("subsequent header should be flush-left, got %q", got)
	}
	if strings.Contains(got, "\r\n  ") || strings.Contains(got, "\r\n\t") {
		t.Fatalf("no line should keep leading indent, got %q", got)
	}

	rendered, err := templ.RenderMessageStrict(got, templ.Context{CallID: "abc"})
	if err != nil {
		t.Fatalf("RenderMessageStrict() error = %v", err)
	}
	final := ensureMessageTerminator(rendered)
	if !strings.HasPrefix(final, "Call-ID: abc\r\n") {
		t.Fatalf("final wire payload should be flush-left and call_id-rendered, got %q", final)
	}
	if !strings.HasSuffix(final, "\r\n\r\n") {
		t.Fatalf("final wire payload should end with \\r\\n\\r\\n, got %q", final)
	}
}
