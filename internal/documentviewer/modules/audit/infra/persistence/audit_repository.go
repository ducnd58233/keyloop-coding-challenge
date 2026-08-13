// Package persistence is the search_audit adapter only (R3).
package persistence

import (
	"context"
	"errors"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
)

var _ app.AccessRecorder = (*AuditRepository)(nil)

// AuditRepository is append-only: no Update or Delete method exists (NFR8).
type AuditRepository struct {
	pool *postgres.Pool
}

// NewAuditRepository writes search_audit only (R3).
func NewAuditRepository(pool *postgres.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Record inserts one row. There is no update or delete path (NFR8).
func (r *AuditRepository) Record(ctx context.Context, e domain.AccessEvent) error {
	_, err := postgres.QuerierFrom(ctx, r.pool).Exec(ctx, `
		INSERT INTO search_audit (
			vin_hash, vin_suffix, actor_id, request_id, trace_id,
			outcome, sources_ok, sources_failed, latency_ms, requested_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, e.VINHash, e.VINSuffix, e.ActorID, e.RequestID, e.TraceID,
		string(e.Outcome), e.SourcesOK, e.SourcesFailed, e.Latency.Milliseconds(), e.RequestedAt)
	if err != nil {
		return errors.New("search audit record failed")
	}
	return nil
}
