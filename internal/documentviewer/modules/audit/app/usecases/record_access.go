// Package usecases holds audit flows with no I/O of their own.
package usecases

import (
	"context"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
)

// RecordAccess hashes the VIN before the recorder sees it (FR8, SPEC.md §8).
type RecordAccess struct {
	rec   app.AccessRecorder
	salt  string
	clock common.Clock
}

// AccessInput carries the raw VIN. Hashing happens inside Execute, not in the caller.
type AccessInput struct {
	VIN           string
	ActorID       string
	RequestID     string
	TraceID       string
	Outcome       domain.Outcome
	SourcesOK     int
	SourcesFailed int
	Latency       time.Duration
}

// NewRecordAccess uses UTC wall time when clock is nil so timestamps match timestamptz.
func NewRecordAccess(rec app.AccessRecorder, salt string, clock common.Clock) *RecordAccess {
	if clock == nil {
		clock = common.SystemClock{}
	}
	return &RecordAccess{rec: rec, salt: salt, clock: clock}
}

// Execute never passes the full VIN to AccessRecorder.
func (u *RecordAccess) Execute(ctx context.Context, in AccessInput) error {
	hash, suffix := common.HashVIN(u.salt, in.VIN)
	return u.rec.Record(ctx, domain.AccessEvent{
		VINHash:       hash,
		VINSuffix:     suffix,
		ActorID:       in.ActorID,
		RequestID:     in.RequestID,
		TraceID:       in.TraceID,
		Outcome:       in.Outcome,
		SourcesOK:     in.SourcesOK,
		SourcesFailed: in.SourcesFailed,
		Latency:       in.Latency,
		RequestedAt:   u.clock.Now(),
	})
}
