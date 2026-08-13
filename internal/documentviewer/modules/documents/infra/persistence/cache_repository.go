// Package persistence is the document_cache adapter only (R3).
package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
)

var _ app.DocumentCache = (*CacheRepository)(nil)

// ErrPartialNotCached is NFR6: a degraded result must never be written.
var ErrPartialNotCached = errors.New("partial result must not be cached")

// CacheRepository never begins a transaction (R4); it joins UnitOfWork via context.
type CacheRepository struct {
	pool  *postgres.Pool
	clock common.Clock
}

type cachePayload struct {
	Documents []domain.Document     `json:"documents"`
	Sources   []domain.SourceReport `json:"sources"`
}

// NewCacheRepository uses UTC wall time when clock is nil so TTL matches timestamptz.
func NewCacheRepository(pool *postgres.Pool, clock common.Clock) *CacheRepository {
	if clock == nil {
		clock = common.SystemClock{}
	}
	return &CacheRepository{pool: pool, clock: clock}
}

// Lookup decides fresh vs stale on read so an expired row still serves FR10.
func (r *CacheRepository) Lookup(ctx context.Context, vin string) (domain.CachedResult, bool, error) {
	var raw []byte
	var expiresAt time.Time
	err := postgres.QuerierFrom(ctx, r.pool).QueryRow(ctx, `
		SELECT payload, expires_at
		FROM document_cache
		WHERE vin = $1
	`, vin).Scan(&raw, &expiresAt)
	if errors.Is(err, postgres.ErrNoRows) {
		return domain.CachedResult{}, false, nil
	}
	if err != nil {
		return domain.CachedResult{}, false, errors.New("document cache lookup failed")
	}

	var payload cachePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return domain.CachedResult{}, false, errors.New("document cache payload invalid")
	}
	if payload.Documents == nil {
		payload.Documents = []domain.Document{}
	}
	if payload.Sources == nil {
		payload.Sources = []domain.SourceReport{}
	}

	return domain.CachedResult{
		Result: domain.AggregateResult{
			Documents: payload.Documents,
			Sources:   payload.Sources,
		},
		Stale: !expiresAt.After(r.clock.Now()),
	}, true, nil
}

// Store refuses Partial so one upstream blip cannot poison the TTL (NFR6).
func (r *CacheRepository) Store(ctx context.Context, vin string, result domain.AggregateResult, ttl time.Duration) error {
	if result.Partial {
		return ErrPartialNotCached
	}
	body, err := json.Marshal(cachePayload{
		Documents: result.Documents,
		Sources:   result.Sources,
	})
	if err != nil {
		return errors.New("document cache encode failed")
	}
	now := r.clock.Now()
	_, err = postgres.QuerierFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO document_cache (vin, payload, source_signature, cached_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (vin) DO UPDATE SET
			payload = EXCLUDED.payload,
			source_signature = EXCLUDED.source_signature,
			cached_at = EXCLUDED.cached_at,
			expires_at = EXCLUDED.expires_at
	`, vin, body, sourceSignature(result), now, now.Add(ttl))
	if err != nil {
		return errors.New("document cache store failed")
	}
	return nil
}

func sourceSignature(r domain.AggregateResult) string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(r.Sources))
	for _, s := range r.Sources {
		if s.Status != domain.SourceStatusOK {
			continue
		}
		n := string(s.Name)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
