// Package domain holds audit types. It imports nothing above itself (R2).
package domain

import "time"

// Outcome is the request result recorded on every lookup (FR8).
type Outcome string

const (
	// OutcomeOK is a complete successful lookup.
	OutcomeOK Outcome = "OK"
	// OutcomePartial is at least one source down with others healthy (FR7).
	OutcomePartial Outcome = "PARTIAL"
	// OutcomeStale is an expired cache served after total upstream failure (FR10).
	OutcomeStale Outcome = "STALE"
	// OutcomeInvalidVIN is a rejected VIN (A1).
	OutcomeInvalidVIN Outcome = "INVALID_VIN"
	// OutcomeUnavailable is total failure with no usable cache.
	OutcomeUnavailable Outcome = "UNAVAILABLE"
)

// AccessEvent is one append-only audit row (DRAFT.md §5.2). No full VIN.
type AccessEvent struct {
	VINHash       string
	VINSuffix     string
	ActorID       string
	RequestID     string
	TraceID       string
	Outcome       Outcome
	SourcesOK     int
	SourcesFailed int
	Latency       time.Duration
	RequestedAt   time.Time
}
