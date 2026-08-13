// Package mockfault injects latency, random 500s, timeouts and full outages into mock upstreams.
package mockfault

import (
	"context"
	"net/http"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/randutil"
)

// Live defaults: most requests succeed; some 500; some hang past the 2s budget;
// some take a random delay that may still beat the per-source timeout, or not.
const (
	DefaultErrorRate   = 0.15
	DefaultTimeoutRate = 0.15
	DefaultTimeoutFor  = 3 * time.Second
	DefaultLatencyRate = 0.20
	DefaultLatencyLow  = 100 * time.Millisecond
	DefaultLatencyHigh = 2500 * time.Millisecond
)

// Kind is the fault bucket chosen for one request.
type Kind string

const (
	// KindOK is a fast success with no injected delay.
	KindOK Kind = "ok"
	// KindDown is a full outage (503).
	KindDown Kind = "down"
	// KindError is a random 500.
	KindError Kind = "error"
	// KindTimeout hangs past the per-source budget and writes no body.
	KindTimeout Kind = "timeout"
	// KindLatency delays then continues to a success body.
	KindLatency Kind = "latency"
)

// Result is what Apply did: kind, delay applied, and whether the handler must stop.
type Result struct {
	Kind    Kind
	Latency time.Duration
	Stop    bool
}

// Config is the T3 fault-injection surface.
type Config struct {
	Latency        time.Duration
	LatencyRate    float64
	LatencyMin     time.Duration
	LatencyMax     time.Duration
	ErrorRate      float64
	TimeoutRate    float64
	TimeoutFor     time.Duration
	Down           bool
	Sleep          func(time.Duration)
	Random         func() float64
	RandomDuration func() time.Duration
}

// Live returns per-request random success / latency / 500 / timeout. Not read from env.
func Live() Config {
	return Config{
		ErrorRate:   DefaultErrorRate,
		TimeoutRate: DefaultTimeoutRate,
		TimeoutFor:  DefaultTimeoutFor,
		LatencyRate: DefaultLatencyRate,
		LatencyMin:  DefaultLatencyLow,
		LatencyMax:  DefaultLatencyHigh,
	}
}

func defaultRandom() float64 {
	v, err := randutil.Float64()
	if err != nil {
		return 1
	}
	return v
}

func (c Config) duration() time.Duration {
	if c.RandomDuration != nil {
		return c.RandomDuration()
	}
	low := c.LatencyMin
	high := c.LatencyMax
	if low <= 0 {
		low = DefaultLatencyLow
	}
	if high <= 0 {
		high = DefaultLatencyHigh
	}
	d, err := randutil.DurationBetween(low, high)
	if err != nil {
		return low
	}
	return d
}

func (c Config) wait(ctx context.Context, d time.Duration) {
	if c.Sleep != nil {
		c.Sleep(d)
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// Apply maybe delays or short-circuits the request. Callers log Result then stop if Stop.
func (c Config) Apply(ctx context.Context, w http.ResponseWriter) Result {
	if c.Down {
		w.WriteHeader(http.StatusServiceUnavailable)
		return Result{Kind: KindDown, Stop: true}
	}

	rnd := c.Random
	if rnd == nil {
		rnd = defaultRandom
	}

	r := rnd()
	if c.TimeoutRate > 0 && r < c.TimeoutRate {
		d := c.TimeoutFor
		if d <= 0 {
			d = DefaultTimeoutFor
		}
		c.wait(ctx, d)
		return Result{Kind: KindTimeout, Latency: d, Stop: true}
	}
	if c.ErrorRate > 0 && r < c.TimeoutRate+c.ErrorRate {
		w.WriteHeader(http.StatusInternalServerError)
		return Result{Kind: KindError, Stop: true}
	}
	if c.LatencyRate > 0 && r < c.TimeoutRate+c.ErrorRate+c.LatencyRate {
		d := c.duration()
		c.wait(ctx, d)
		return Result{Kind: KindLatency, Latency: d}
	}
	if c.Latency > 0 {
		c.wait(ctx, c.Latency)
		return Result{Kind: KindLatency, Latency: c.Latency}
	}
	return Result{Kind: KindOK}
}

// Intercept is Apply().Stop for call sites that only need the short-circuit bit.
func (c Config) Intercept(w http.ResponseWriter) bool {
	return c.Apply(context.Background(), w).Stop
}
