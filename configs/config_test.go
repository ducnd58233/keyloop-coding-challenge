package configs

import (
	"strings"
	"testing"
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

func TestFloatEnv(t *testing.T) {
	t.Setenv("RATE", "0.25")
	got, err := float("RATE", 0)
	if err != nil || got != 0.25 {
		t.Fatalf("float(RATE) = %v, %v, want 0.25", got, err)
	}
	got, err = float("RATE_MISSING", 0.5)
	if err != nil || got != 0.5 {
		t.Fatalf("float missing fallback = %v, %v, want 0.5", got, err)
	}
}

func TestBooleanEnv(t *testing.T) {
	t.Setenv("FLAG_ON", "true")
	got, err := boolean("FLAG_ON", false)
	if err != nil || !got {
		t.Fatalf("boolean(FLAG_ON) = %v, %v, want true", got, err)
	}
	got, err = boolean("FLAG_MISSING", true)
	if err != nil || !got {
		t.Fatalf("boolean missing fallback = %v, %v, want true", got, err)
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

func TestLoadDefaultHTTPAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("PER_SOURCE_TIMEOUT", "2s")
	t.Setenv("AGGREGATE_TIMEOUT", "2500ms")
	t.Setenv("REQUEST_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Address != ":8000" {
		t.Fatalf("HTTP.Address = %q, want :8000", cfg.HTTP.Address)
	}
}
