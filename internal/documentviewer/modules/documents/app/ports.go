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

// CacheStore must not persist partial aggregates (NFR6).
type CacheStore interface {
	Lookup(ctx context.Context, vin string) (domain.CachedResult, bool, error)
	Store(ctx context.Context, vin string, r domain.AggregateResult, ttl time.Duration) error
}
