package domain

import "errors"

// Wire codes are FR7 vocabulary, not HTTP status text.
const (
	CodeInvalidVIN            = "VIN_INVALID"
	CodeAllSourcesUnavailable = "ALL_SOURCES_UNAVAILABLE"
	CodeUpstreamTimeout       = "UPSTREAM_TIMEOUT"
	CodeUpstreamError         = "UPSTREAM_ERROR"
)

var (
	// ErrInvalidVIN is A1: callers map it to 400, not a source failure.
	ErrInvalidVIN = errors.New("invalid vin")
	// ErrAllSourcesUnavailable is returned only when every source failed and no stale cache exists.
	ErrAllSourcesUnavailable = errors.New("all sources unavailable")
)
