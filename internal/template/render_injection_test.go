package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderFieldUsesDefaultInjectionFile(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "data.csv")
	// SIPp-style header row then semicolon-separated data.
	if err := os.WriteFile(csvPath, []byte("SEQUENTIAL\nalpha;beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := Context{
		BasePath:          dir,
		CallNumber:        2,
		InjectionFile:     csvPath,
		CSVFieldOverrides: make(map[string]map[int]map[int]string),
	}
	got, err := ctx.RenderStrict("[field0]")
	if err != nil {
		t.Fatal(err)
	}
	if got != "alpha" {
		t.Fatalf("field0: got %q want alpha", got)
	}
	got2, err := ctx.RenderStrict("[field1]")
	if err != nil {
		t.Fatal(err)
	}
	if got2 != "beta" {
		t.Fatalf("field1: got %q want beta", got2)
	}
}

func TestRenderFieldExplicitFileOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	def := filepath.Join(dir, "default.csv")
	other := filepath.Join(dir, "other.csv")
	if err := os.WriteFile(def, []byte("wrong\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("good,csv\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := Context{
		BasePath:          dir,
		CallNumber:        1,
		InjectionFile:     def,
		CSVFieldOverrides: make(map[string]map[int]map[int]string),
	}
	got, err := ctx.RenderStrict("[field0 file=other.csv]")
	if err != nil {
		t.Fatal(err)
	}
	if got != "good" {
		t.Fatalf("got %q want good", got)
	}
}

func TestRenderFieldWithoutInjectionFileUnresolvedStrict(t *testing.T) {
	ctx := Context{CallNumber: 1, BasePath: t.TempDir()}
	_, err := ctx.RenderStrict("[field0]")
	if err == nil {
		t.Fatal("expected error for [field0] without -inf default")
	}
}

func TestRenderFieldBeyondRowWidthReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "onecol.csv")
	if err := os.WriteFile(csvPath, []byte("only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := Context{
		BasePath:          dir,
		CallNumber:        1,
		InjectionFile:     csvPath,
		CSVFieldOverrides: make(map[string]map[int]map[int]string),
	}
	got0, err := ctx.RenderStrict("[field0]")
	if err != nil {
		t.Fatal(err)
	}
	if got0 != "only" {
		t.Fatalf("field0: got %q", got0)
	}
	got1, err := ctx.RenderStrict("[field1]")
	if err != nil {
		t.Fatalf("[field1] should resolve to empty when row has one column: %v", err)
	}
	if got1 != "" {
		t.Fatalf("field1: got %q want empty", got1)
	}
}
