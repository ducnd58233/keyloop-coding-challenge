// Package app holds documents use cases and the ports they depend on (R1).
//
//go:generate mockgen -source=$GOFILE -destination=mocks/mock_$GOFILE -package=${GOPACKAGE}mocks
package app

import (
	"context"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

// DocumentSource is one upstream back-office system (SYSTEM_DESIGN §3.2).
type DocumentSource interface {
	Name() domain.SourceName
	Fetch(ctx context.Context, vin string) ([]domain.Document, error)
}

// CacheStore is the TTL document cache (FR9, FR10, NFR6, NFR7).
type CacheStore interface {
	Lookup(ctx context.Context, vin string) (domain.CachedResult, bool, error)
	Store(ctx context.Context, vin string, r domain.AggregateResult, ttl time.Duration) error
}
