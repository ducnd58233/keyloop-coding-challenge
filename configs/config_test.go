package configs

import (
	"strings"
	"testing"
	"time"
)

func TestLoadTimeoutOrdering(t *testing.T) {
	tests := []struct {
		name       string
		perSource  string
		aggregate  string
		request    string
		wantErr    bool
		errContain string
	}{
		{name: "spec defaults", perSource: "2s", aggregate: "2500ms", request: "3s"},
		{name: "valid custom budget", perSource: "1s", aggregate: "2s", request: "4s"},
		{name: "per-source equals aggregate", perSource: "2s", aggregate: "2s", request: "3s", wantErr: true, errContain: "PER_SOURCE_TIMEOUT"},
		{name: "per-source greater than aggregate", perSource: "3s", aggregate: "2s", request: "4s", wantErr: true, errContain: "PER_SOURCE_TIMEOUT"},
		{name: "aggregate equals request", perSource: "1s", aggregate: "3s", request: "3s", wantErr: true, errContain: "AGGREGATE_TIMEOUT"},
		{name: "aggregate greater than request", perSource: "1s", aggregate: "4s", request: "3s", wantErr: true, errContain: "AGGREGATE_TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PER_SOURCE_TIMEOUT", tt.perSource)
			t.Setenv("AGGREGATE_TIMEOUT", tt.aggregate)
			t.Setenv("REQUEST_TIMEOUT", tt.request)

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want substring %q", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("Load() error = %q, want substring %q", err.Error(), tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.Sources.PerSourceTimeout >= cfg.Sources.AggregateTimeout {
				t.Fatalf("per-source %s not less than aggregate %s",
					cfg.Sources.PerSourceTimeout, cfg.Sources.AggregateTimeout)
			}
			if cfg.Sources.AggregateTimeout >= cfg.HTTP.RequestTimeout {
				t.Fatalf("aggregate %s not less than request %s",
					cfg.Sources.AggregateTimeout, cfg.HTTP.RequestTimeout)
			}
		})
	}
}

func TestLoadInvalidEnv(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "bad request timeout", key: "REQUEST_TIMEOUT", value: "abc"},
		{name: "bare number duration", key: "PER_SOURCE_TIMEOUT", value: "2"},
		{name: "bad db max conns", key: "DB_MAX_CONNS", value: "nope"},
		{name: "bad mock error rate", key: "MOCK_ERROR_RATE", value: "bad"},
		{name: "bad mock down", key: "MOCK_DOWN", value: "yes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PER_SOURCE_TIMEOUT", "2s")
			t.Setenv("AGGREGATE_TIMEOUT", "2500ms")
			t.Setenv("REQUEST_TIMEOUT", "3s")
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want parse error")
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Load() error = %q, want substring %q", err.Error(), tt.key)
			}
		})
	}
}

func TestLoadTypedMockFlags(t *testing.T) {
	t.Setenv("PER_SOURCE_TIMEOUT", "2s")
	t.Setenv("AGGREGATE_TIMEOUT", "2500ms")
	t.Setenv("REQUEST_TIMEOUT", "3s")
	t.Setenv("MOCK_LATENCY_MS", "150")
	t.Setenv("MOCK_ERROR_RATE", "0.25")
	t.Setenv("MOCK_DOWN", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Sources.MockLatency != 150*time.Millisecond {
		t.Fatalf("MockLatency = %s, want 150ms", cfg.Sources.MockLatency)
	}
	if cfg.Sources.MockErrorRate != 0.25 {
		t.Fatalf("MockErrorRate = %v, want 0.25", cfg.Sources.MockErrorRate)
	}
	if !cfg.Sources.MockDown {
		t.Fatal("MockDown = false, want true")
	}
}
