// Package domain holds document types and merge rules. It imports nothing above itself (R2).
package domain

import "time"

// Document is the unified metadata+URL model (A4).
type Document struct {
	ID       string
	Source   SourceName
	Type     string
	Title    string
	IssuedAt time.Time
	URL      string
}

// AggregateResult is FR7: useful when one upstream is down.
type AggregateResult struct {
	Documents       []Document
	Sources         []SourceReport
	Partial         bool
	Stale           bool
	ServedFromCache bool
}

// CachedResult is a previously stored aggregate, possibly past TTL (FR10).
type CachedResult struct {
	Result AggregateResult
	Stale  bool
}
