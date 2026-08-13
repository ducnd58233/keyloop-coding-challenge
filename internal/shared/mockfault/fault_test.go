package mockfault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInterceptDown(t *testing.T) {
	rec := httptest.NewRecorder()
	cfg := Config{Down: true}
	if !cfg.Intercept(rec) {
		t.Fatal("Intercept() = false, want true when Down")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestInterceptErrorRateHit(t *testing.T) {
	rec := httptest.NewRecorder()
	cfg := Config{ErrorRate: 1, Random: func() float64 { return 0 }}
	if !cfg.Intercept(rec) {
		t.Fatal("Intercept() = false, want true when Random < ErrorRate")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestInterceptErrorRateMiss(t *testing.T) {
	rec := httptest.NewRecorder()
	cfg := Config{ErrorRate: 0.5, Random: func() float64 { return 0.9 }}
	if cfg.Intercept(rec) {
		t.Fatal("Intercept() = true, want false when Random >= ErrorRate")
	}
	if rec.Code != http.StatusOK && rec.Code != 0 {
		t.Fatalf("status = %d, want unset", rec.Code)
	}
}

func TestInterceptTimeout(t *testing.T) {
	var slept time.Duration
	rec := httptest.NewRecorder()
	cfg := Config{
		TimeoutRate: 1,
		TimeoutFor:  3 * time.Second,
		Random:      func() float64 { return 0 },
		Sleep:       func(d time.Duration) { slept = d },
	}
	if !cfg.Intercept(rec) {
		t.Fatal("Intercept() = false, want true on timeout")
	}
	if slept != 3*time.Second {
		t.Fatalf("slept = %s, want 3s", slept)
	}
	if rec.Code == http.StatusInternalServerError || rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("status = %d, timeout must not write 5xx (client deadline fires)", rec.Code)
	}
}

func TestInterceptLatency(t *testing.T) {
	var slept time.Duration
	rec := httptest.NewRecorder()
	cfg := Config{
		Latency: 150 * time.Millisecond,
		Sleep:   func(d time.Duration) { slept = d },
	}
	if cfg.Intercept(rec) {
		t.Fatal("Intercept() = true, want false")
	}
	if slept != 150*time.Millisecond {
		t.Fatalf("slept = %s, want 150ms", slept)
	}
}

func TestLiveDefaults(t *testing.T) {
	cfg := Live()
	if cfg.ErrorRate != DefaultErrorRate || cfg.TimeoutRate != DefaultTimeoutRate {
		t.Fatalf("Live() = %+v, want error/timeout rates set", cfg)
	}
	if cfg.LatencyRate != DefaultLatencyRate || cfg.LatencyMin != DefaultLatencyLow || cfg.LatencyMax != DefaultLatencyHigh {
		t.Fatalf("Live() = %+v, want latency band set", cfg)
	}
	if cfg.Down {
		t.Fatal("Live() must not force Down; use -down for demo-degraded")
	}
}

func TestApplyRandomLatency(t *testing.T) {
	var slept time.Duration
	rec := httptest.NewRecorder()
	want := 400 * time.Millisecond
	cfg := Config{
		LatencyRate:    1,
		Random:         func() float64 { return 0 },
		Sleep:          func(d time.Duration) { slept = d },
		RandomDuration: func() time.Duration { return want },
	}
	got := cfg.Apply(context.Background(), rec)
	if got.Stop {
		t.Fatal("Apply() Stop = true, want false for latency-then-success")
	}
	if got.Kind != KindLatency {
		t.Fatalf("Kind = %s, want latency", got.Kind)
	}
	if got.Latency != want || slept != want {
		t.Fatalf("latency = %s slept = %s, want %s", got.Latency, slept, want)
	}
	if rec.Code == http.StatusInternalServerError || rec.Code == http.StatusServiceUnavailable {
		t.Fatalf("status = %d, latency must not write 5xx", rec.Code)
	}
}

func TestApplyBuckets(t *testing.T) {
	cases := []struct {
		name string
		r    float64
		kind Kind
		stop bool
	}{
		{name: "timeout", r: 0.05, kind: KindTimeout, stop: true},
		{name: "error", r: 0.20, kind: KindError, stop: true},
		{name: "latency", r: 0.40, kind: KindLatency, stop: false},
		{name: "ok", r: 0.90, kind: KindOK, stop: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cfg := Config{
				TimeoutRate:    0.15,
				ErrorRate:      0.15,
				LatencyRate:    0.20,
				Random:         func() float64 { return tc.r },
				Sleep:          func(time.Duration) {},
				RandomDuration: func() time.Duration { return 50 * time.Millisecond },
			}
			got := cfg.Apply(context.Background(), rec)
			if got.Kind != tc.kind || got.Stop != tc.stop {
				t.Fatalf("Apply = %+v, want kind=%s stop=%v", got, tc.kind, tc.stop)
			}
		})
	}
}
