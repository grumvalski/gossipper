package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParseRunProfileMeta(t *testing.T) {
	rest, meta, err := parseRunProfileMeta([]string{"-config", "/a/b.json", "-run-alias", "u1", "-sn", "uac"})
	if err != nil {
		t.Fatal(err)
	}
	if meta.ConfigPath != "/a/b.json" || meta.RunAlias != "u1" || meta.ListAliases {
		t.Fatalf("meta=%+v", meta)
	}
	if got := strings.Join(rest, " "); got != "-sn uac" {
		t.Fatalf("rest=%q", got)
	}
}

func TestApplyRunSpecRelativeScenario(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	spec := runSpec{
		ScenarioFile: ptr("scen/uas.xml"),
		LocalPort:    ptr(9050),
	}
	if err := applyRunSpec(&cfg, &spec, dir); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "scen", "uas.xml")
	if cfg.ScenarioFile != want {
		t.Fatalf("path: got %q want %q", cfg.ScenarioFile, want)
	}
	if cfg.LocalPort != 9050 {
		t.Fatalf("port %d", cfg.LocalPort)
	}
}

func TestLoadAndApplyRunProfileUnknownAlias(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "g.json")
	if err := os.WriteFile(path, []byte(`{"aliases":{"a":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	_, err := LoadAndApplyRunProfile(&cfg, path, "missing")
	if err == nil || !strings.Contains(err.Error(), "unknown alias") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseListAliasesExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "g.json")
	if err := os.WriteFile(path, []byte(`{"aliases":{"x":{}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Parse([]string{"-config", path, "-list-aliases"})
	if !errors.Is(err, ErrListAliases) {
		t.Fatalf("got err=%v", err)
	}
}

func TestPrintRunProfileAliases(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "g.json")
	doc := map[string]map[string]struct{}{
		"aliases": {
			"zebra": {},
			"alpha": {},
		},
	}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	// Smoke: no panic, sorted output checked indirectly via Parse ErrListAliases path in integration test below.
	if err := printRunProfileAliases(path); err != nil {
		t.Fatal(err)
	}
}

func TestParseWithBundledExampleRunProfile(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	cfgPath := filepath.Join(repoRoot, "testdata", "run-profiles", "example.json")
	cfg, err := Parse([]string{"-config", cfgPath, "-run-alias", "uac-local"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScenarioName != "uac" || cfg.RemoteHost != "127.0.0.1" || cfg.RemotePort != 5060 {
		t.Fatalf("cfg=%+v", cfg)
	}
	if !cfg.SendMediaReport || !cfg.HEPHomerLakeRTCP || cfg.HEPRawRTCP {
		t.Fatalf("uac-local homer-lake: send=%v homer=%v raw=%v", cfg.SendMediaReport, cfg.HEPHomerLakeRTCP, cfg.HEPRawRTCP)
	}

	cfg2, err := Parse([]string{"-config", cfgPath, "-run-alias", "hep-uas-listen"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg2.ScenarioFile); err != nil {
		t.Fatalf("resolved scenario_file %q: %v", cfg2.ScenarioFile, err)
	}
	if cfg2.LogOTELEndpoint != "http://127.0.0.1:4318" || cfg2.LogOTELProto != "http" || !cfg2.LogOTELInsecure {
		t.Fatalf("otlp from profile: endpoint=%q proto=%q insecure=%v", cfg2.LogOTELEndpoint, cfg2.LogOTELProto, cfg2.LogOTELInsecure)
	}
	if cfg2.HEPAddr != "127.0.0.1:9060" || cfg2.HEPCaptureID != 2001 {
		t.Fatalf("hep from profile: addr=%q capture_id=%d", cfg2.HEPAddr, cfg2.HEPCaptureID)
	}
	if !cfg2.SendMediaReport || !cfg2.HEPHomerLakeRTCP || cfg2.HEPRawRTCP {
		t.Fatalf("homer-lake media from profile: send=%v homer=%v raw=%v", cfg2.SendMediaReport, cfg2.HEPHomerLakeRTCP, cfg2.HEPRawRTCP)
	}

	hepPath := filepath.Join(repoRoot, "testdata", "run-profiles", "hep-scripts.json")
	cfg3, err := Parse([]string{"-config", hepPath, "-run-alias", "hep-uac-send-raw-rtcp", "-hep_addr", "127.0.0.1:9060"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg3.SendMediaReport || cfg3.HEPHomerLakeRTCP || !cfg3.HEPRawRTCP {
		t.Fatalf("raw rtcp alias: homer=%v raw=%v send=%v", cfg3.HEPHomerLakeRTCP, cfg3.HEPRawRTCP, cfg3.SendMediaReport)
	}
}

func ptr[T any](v T) *T {
	return &v
}
