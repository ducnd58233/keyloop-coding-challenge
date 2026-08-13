// Package common holds clock and salted VIN hashing shared across modules.
package common

import (
	"sync"
	"time"
)

// Clock is injectable so TTL tests do not sleep (FR9, FR10).
type Clock interface {
	Now() time.Time
}

// SystemClock is UTC wall time so cache expiry does not depend on local TZ.
type SystemClock struct{}

// Now uses UTC so expiry comparisons match timestamptz.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// FixedClock is for tests. Advance moves Now without sleeping.
type FixedClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFixedClock stores t in UTC so fake expiry matches timestamptz.
func NewFixedClock(t time.Time) *FixedClock {
	return &FixedClock{now: t.UTC()}
}

// Now is mutex-guarded so Advance is safe under parallel tests.
func (c *FixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance expires a cache row without a wall-clock wait.
func (c *FixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
