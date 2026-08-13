package app

import (
	"context"
	"errors"
	"time"

	audituc "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app/usecases"
	auditdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	docsapi "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/api"
	docsdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

const auditWriteTimeout = 2 * time.Second

var _ docsapi.AccessLog = (*accessLog)(nil)

// accessLog adapts RecordAccess to the documents API port so api never imports audit (R1).
type accessLog struct {
	rec *audituc.RecordAccess
	log observability.Logger
}

func newAccessLog(rec *audituc.RecordAccess, log observability.Logger) *accessLog {
	return &accessLog{rec: rec, log: log}
}

// Record hashes inside RecordAccess. A cancelled request ctx must not drop FR8.
func (a *accessLog) Record(ctx context.Context, vin, actorID, requestID string, result docsdomain.AggregateResult, err error, latency time.Duration) {
	if a == nil || a.rec == nil {
		return
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	defer cancel()

	ok, failed := 0, 0
	for _, s := range result.Sources {
		if s.Status == docsdomain.SourceStatusOK {
			ok++
			continue
		}
		failed++
	}

	if recErr := a.rec.Execute(auditCtx, audituc.AccessInput{
		VIN:           vin,
		ActorID:       actorID,
		RequestID:     requestID,
		Outcome:       auditOutcome(result, err),
		SourcesOK:     ok,
		SourcesFailed: failed,
		Latency:       latency,
	}); recErr != nil && a.log != nil {
		a.log.ErrorContext(auditCtx, "audit record failed", "error", recErr.Error())
	}
}

func auditOutcome(result docsdomain.AggregateResult, err error) auditdomain.Outcome {
	switch {
	case errors.Is(err, docsdomain.ErrInvalidVIN):
		return auditdomain.OutcomeInvalidVIN
	case errors.Is(err, docsdomain.ErrAllSourcesUnavailable), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return auditdomain.OutcomeUnavailable
	case result.Stale:
		return auditdomain.OutcomeStale
	case result.Partial:
		return auditdomain.OutcomePartial
	default:
		return auditdomain.OutcomeOK
	}
}
