// Package domain holds audit types. It imports nothing above itself (R2).
package domain

import "time"

// Outcome is the request result recorded on every lookup (FR8).
type Outcome string

// Outcome values are FR8 audit vocabulary, not HTTP status.
const (
	OutcomeOK          Outcome = "OK"
	OutcomePartial     Outcome = "PARTIAL"     // FR7
	OutcomeStale       Outcome = "STALE"       // FR10
	OutcomeInvalidVIN  Outcome = "INVALID_VIN" // A1
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
