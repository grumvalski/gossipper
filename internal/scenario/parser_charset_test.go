package scenario

import (
	"bytes"
	"strings"
	"testing"
)

// In Go, string(byte(0xE9)) is UTF-8 for U+00E9 (two bytes), not Latin-1 0xE9.
// Use "\xe9" in a double-quoted string for a single Latin-1 byte in test XML.

func TestScenarioXMLToUTF8RewritesDecl(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0" encoding="ISO-8859-1"?>` + "\n" +
		`<scenario name="caf` + "\xe9" + `">` + "\n" +
		`  <send><![CDATA[OPTIONS sip:u@h SIP/2.0]]></send>` + "\n" +
		`</scenario>`

	out, err := scenarioXMLToUTF8([]byte(xml))
	if err != nil {
		t.Fatalf("scenarioXMLToUTF8: %v", err)
	}
	if !bytes.Contains(out, []byte(`encoding="UTF-8"`)) {
		p := min(120, len(out))
		t.Fatalf("expected UTF-8 encoding in prolog, got prefix: %s", string(out[:p]))
	}
	if bytes.Contains(out, []byte(`encoding="ISO-8859-1"`)) {
		t.Fatal("prolog still declares ISO-8859-1 after rewrite")
	}
}

func TestScenarioXMLToUTF8NameBytes(t *testing.T) {
	t.Parallel()
	xml := `<?xml version="1.0" encoding="ISO-8859-1"?>` + "\n" +
		`<scenario name="caf` + "\xe9" + `">` + "\n" +
		`  <send><![CDATA[OK]]></send>` + "\n" + `</scenario>`
	if !bytes.Contains([]byte(xml), []byte{0xe9}) {
		t.Fatalf("input should contain Latin-1 0xE9; got % x", []byte(xml))
	}
	out, err := scenarioXMLToUTF8([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	i := bytes.Index(out, []byte(`name="`))
	if i < 0 {
		t.Fatalf("no name=: %s", out)
	}
	start := i + len(`name="`)
	end := bytes.IndexByte(out[start:], '"')
	if end < 0 {
		t.Fatal("no closing quote")
	}
	val := out[start : start+end]
	want := []byte("café")
	if !bytes.Equal(val, want) {
		t.Fatalf("name bytes % x want % x", val, want)
	}
}

func TestParseStringISO8859_1Prolog(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0" encoding="ISO-8859-1"?>` + "\n" +
		`<scenario name="caf` + "\xe9" + `">` + "\n" +
		`  <send><![CDATA[OPTIONS sip:u@h SIP/2.0]]></send>` + "\n" +
		`</scenario>`

	sc, err := ParseString(xml)
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}
	if want := "café"; sc.Name != want {
		t.Fatalf("scenario name = %q, want %q", sc.Name, want)
	}
	if len(sc.Commands) != 1 || sc.Commands[0].Type != CommandSend {
		t.Fatalf("unexpected commands: %+v", sc.Commands)
	}
	if !strings.Contains(sc.Commands[0].SendText, "OPTIONS") {
		t.Fatalf("send body: %q", sc.Commands[0].SendText)
	}
}

func TestParseStringUTF8StillWorks(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<scenario name="plain">` + "\n" +
		`  <send><![CDATA[OPTIONS sip:u@h SIP/2.0]]></send>` + "\n" +
		`</scenario>`

	sc, err := ParseString(xml)
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}
	if sc.Name != "plain" {
		t.Fatalf("name = %q", sc.Name)
	}
}
