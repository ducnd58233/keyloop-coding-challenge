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

// AggregateResult is what the use case returns after fan-out, merge and sort.
type AggregateResult struct {
	Documents []Document
	Partial   bool
}

// CachedResult is a previously stored aggregate, possibly past TTL (FR10).
type CachedResult struct {
	Result AggregateResult
	Stale  bool
}
