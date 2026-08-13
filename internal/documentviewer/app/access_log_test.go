package app

import (
	"context"
	"errors"
	"testing"

	auditdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	docsdomain "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

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
