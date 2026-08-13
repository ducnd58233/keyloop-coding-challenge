package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	audituc "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app/usecases"
	auditdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	docsdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

const testVIN = "1HGCM82633"

type captureRecorder struct {
	n       int
	ctxDone bool
	ev      auditdomain.AccessEvent
}

func (c *captureRecorder) Record(ctx context.Context, ev auditdomain.AccessEvent) error {
	c.n++
	c.ctxDone = ctx.Err() != nil
	c.ev = ev
	return nil
}

func TestAuditOutcome(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		result docsdomain.AggregateResult
		err    error
		want   auditdomain.Outcome
	}{
		{name: "invalid vin", err: docsdomain.ErrInvalidVIN, want: auditdomain.OutcomeInvalidVIN},
		{name: "all down", err: docsdomain.ErrAllSourcesUnavailable, want: auditdomain.OutcomeUnavailable},
		{name: "timeout", err: context.DeadlineExceeded, want: auditdomain.OutcomeUnavailable},
		{name: "stale", result: docsdomain.AggregateResult{Stale: true}, want: auditdomain.OutcomeStale},
		{name: "partial", result: docsdomain.AggregateResult{Partial: true}, want: auditdomain.OutcomePartial},
		{name: "ok", result: docsdomain.AggregateResult{}, want: auditdomain.OutcomeOK},
		{name: "cache hit", result: docsdomain.AggregateResult{ServedFromCache: true}, want: auditdomain.OutcomeOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := auditOutcome(tc.result, tc.err); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAuditOutcomePrefersInvalidVINOverStaleFlags(t *testing.T) {
	t.Parallel()
	got := auditOutcome(docsdomain.AggregateResult{Stale: true, Partial: true}, docsdomain.ErrInvalidVIN)
	if got != auditdomain.OutcomeInvalidVIN {
		t.Fatalf("got %s", got)
	}
	if errors.Is(docsdomain.ErrInvalidVIN, docsdomain.ErrAllSourcesUnavailable) {
		t.Fatal("sentinels must stay distinct")
	}
}

func TestAccessLogRecordSurvivesCancelledRequest(t *testing.T) {
	t.Parallel()
	rec := &captureRecorder{}
	a := newAccessLog(audituc.NewRecordAccess(rec, "test-salt", nil), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a.Record(ctx, testVIN, "actor", "req-1", docsdomain.AggregateResult{}, context.DeadlineExceeded, time.Millisecond)
	if rec.n != 1 {
		t.Fatalf("recorder calls = %d, want 1", rec.n)
	}
	if rec.ctxDone {
		t.Fatal("FR8 audit ctx must not be the cancelled request ctx")
	}
	if rec.ev.Outcome != auditdomain.OutcomeUnavailable {
		t.Fatalf("outcome = %s", rec.ev.Outcome)
	}
	if rec.ev.VINHash == "" || rec.ev.VINSuffix != "2633" {
		t.Fatalf("hash/suffix = %q %q", rec.ev.VINHash, rec.ev.VINSuffix)
	}
	if strings.Contains(rec.ev.VINHash, testVIN) || rec.ev.VINSuffix == testVIN {
		t.Fatalf("full VIN reached recorder: %+v", rec.ev)
	}
}
