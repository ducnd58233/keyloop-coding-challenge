package domain

import "errors"

// Wire codes on sources[].error (FR7). HTTP envelope errors use status, not these labels.
const (
	CodeUpstreamTimeout = "UPSTREAM_TIMEOUT"
	CodeUpstreamError   = "UPSTREAM_ERROR"
)

var (
	// ErrInvalidVIN is A1: callers map it to 400, not a source failure.
	ErrInvalidVIN = errors.New("invalid vin")
	// ErrAllSourcesUnavailable is returned only when every source failed and no stale cache exists.
	ErrAllSourcesUnavailable = errors.New("all sources unavailable")
)
