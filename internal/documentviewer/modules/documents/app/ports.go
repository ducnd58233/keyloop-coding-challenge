// Package app holds documents use cases and the ports they depend on (R1).
//
//go:generate mockgen -source=$GOFILE -destination=mocks/mock_$GOFILE -package=${GOPACKAGE}mocks
package app

import (
	"context"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

// DocumentSource is one upstream; failures must not cancel the sibling (NFR1).
type DocumentSource interface {
	Name() domain.SourceName
	Fetch(ctx context.Context, vin string) ([]domain.Document, error)
}

// DocumentCache is the Postgres TTL document_cache (NFR6/NFR7). Not Redis, not in-memory.
type DocumentCache interface {
	Lookup(ctx context.Context, vin string) (domain.CachedResult, bool, error)
	Store(ctx context.Context, vin string, r domain.AggregateResult, ttl time.Duration) error
}
