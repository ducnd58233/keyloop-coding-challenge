package observability

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoggerWritesTintedConsoleAndJSONFile(t *testing.T) {
	dir := t.TempDir()
	var console bytes.Buffer
	log, closer, err := NewLogger(Options{
		Service: "sales",
		Level:   "info",
		Stdout:  &console,
		Dir:     dir,
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Info("mock request", "vin_suffix", "WXYZ", "fault", "latency", "latency", "400ms")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	out := console.String()
	if out == "" {
		t.Fatal("console log is empty")
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("console should be tinted text, got JSON: %s", out)
	}
	if !strings.Contains(out, "mock request") || !strings.Contains(out, "WXYZ") || !strings.Contains(out, "sales") {
		t.Fatalf("console missing fields: %s", out)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "sales.log"))
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("file is not JSON: %s\nerr: %v", raw, err)
	}
	if rec["msg"] != "mock request" {
		t.Fatalf("file msg = %#v, want mock request", rec["msg"])
	}
	if rec["name"] != "sales" {
		t.Fatalf("file name = %#v, want sales", rec["name"])
	}
	if rec["vin_suffix"] != "WXYZ" || rec["fault"] != "latency" {
		t.Fatalf("file missing fields: %#v", rec)
	}
}

func TestNewLoggerRequiresService(t *testing.T) {
	_, _, err := NewLogger(Options{Level: "info", Dir: t.TempDir(), Stdout: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error for empty service")
	}
	_, _, err = NewLogger(Options{Service: "../etc", Level: "info", Dir: t.TempDir(), Stdout: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error for path-like service")
	}
}
