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

// Kind is the Apply outcome written to mock logs.
type Kind string

// Kind values match log field "fault".
const (
	KindOK      Kind = "ok"
	KindDown    Kind = "down"
	KindError   Kind = "error"
	KindTimeout Kind = "timeout" // hang; no body. Client timeout is the bound.
	KindLatency Kind = "latency"
)

// Result tells the handler whether to stop writing a success body.
type Result struct {
	Kind    Kind
	Latency time.Duration
	Stop    bool
}

// Config Sleep and Random are test hooks so chaos tests do not wait on real time.
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

// Live is per-request chaos. Not read from the environment.
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

// Apply records the fault as data; a timeout must not cancel the sibling source (NFR1).
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
