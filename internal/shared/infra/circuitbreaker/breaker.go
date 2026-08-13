// Package circuitbreaker fail-fasts a single upstream. One Breaker per source
// name so an open sales breaker cannot cancel or trip service (NFR1 / DD-2).
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// ErrOpen means the breaker is open; callers map it to an upstream error.
var ErrOpen = errors.New("circuit open")

type state int

const (
	closed state = iota
	open
	halfOpen
)

func (s state) String() string {
	switch s {
	case open:
		return "open"
	case halfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

const (
	defaultThreshold = 5
	defaultCooldown  = 30 * time.Second
)

// Settings are per-source. Name is "sales" or "service", never a hostname.
type Settings struct {
	Name      string
	Threshold int
	Cooldown  time.Duration
	Clock     common.Clock
	Log       observability.Logger
}

// Breaker is safe for concurrent Execute. Timeouts trip it; client cancel does not.
type Breaker struct {
	name      string
	threshold int
	cooldown  time.Duration
	clock     common.Clock
	log       observability.Logger
	mu        sync.Mutex
	state     state
	failures  int
	openUntil time.Time
}

// New applies defaults when Threshold or Cooldown are zero.
func New(s Settings) *Breaker {
	th := s.Threshold
	if th <= 0 {
		th = defaultThreshold
	}
	cd := s.Cooldown
	if cd <= 0 {
		cd = defaultCooldown
	}
	clk := s.Clock
	if clk == nil {
		clk = common.SystemClock{}
	}
	return &Breaker{
		name:      s.Name,
		threshold: th,
		cooldown:  cd,
		clock:     clk,
		log:       s.Log,
	}
}

// Execute runs fn when the circuit allows it. fn must not log a VIN.
func (b *Breaker) Execute(fn func() error) error {
	if b == nil {
		return fn()
	}
	if err := b.allow(); err != nil {
		return err
	}
	err := fn()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			b.releaseProbe()
			return err
		}
		b.fail()
		return err
	}
	b.succeed()
	return nil
}

func (b *Breaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case closed:
		return nil
	case open:
		if b.clock.Now().Before(b.openUntil) {
			return ErrOpen
		}
		b.setState(halfOpen)
		return nil
	default: // halfOpen
		return ErrOpen
	}
}

func (b *Breaker) succeed() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.setState(closed)
}

func (b *Breaker) fail() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == halfOpen {
		b.trip()
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.trip()
	}
}

func (b *Breaker) releaseProbe() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == halfOpen {
		b.openUntil = b.clock.Now()
		b.setState(open)
	}
}

func (b *Breaker) trip() {
	b.openUntil = b.clock.Now().Add(b.cooldown)
	b.setState(open)
}

func (b *Breaker) setState(next state) {
	if b.state == next {
		return
	}
	prev := b.state
	b.state = next
	if b.log == nil {
		return
	}
	b.log.Info("circuit state",
		"breaker", b.name,
		"from", prev.String(),
		"to", next.String(),
	)
}
