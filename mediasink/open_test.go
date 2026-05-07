package mediasink

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildHomerLakeRTCPJSONShape(t *testing.T) {
	t.Parallel()

	state := &rtpStreamState{
		packetCount:      137,
		octetCount:       21920,
		lastRTPTimestamp: 0xdeadbeef,
		ntpMSW:           3373905462,
		ntpLSW:           4286124543,
	}
	raw, err := buildHomerLakeRTCPJSON(0x6c3e4f52, state, time.Unix(1700000000, 0).UTC())
	if err != nil {
		t.Fatalf("buildHomerLakeRTCPJSON() error = %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, key := range []string{"type", "ssrc", "report_count", "report_blocks", "sender_information"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, string(raw))
		}
	}
	if !strings.Contains(string(raw), `"type":200`) {
		t.Fatalf("expected type 200 SR, got %s", string(raw))
	}
}
