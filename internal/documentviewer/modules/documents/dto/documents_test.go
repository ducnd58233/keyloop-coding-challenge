package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

func TestFromAggregateSnakeCaseSixA4Fields(t *testing.T) {
	t.Parallel()
	issued := time.Date(2025, 1, 4, 9, 30, 0, 0, time.UTC)
	got := FromAggregate("1HGCM82633", "req-1", domain.AggregateResult{
		Documents: []domain.Document{{
			ID:       "service:WO-77",
			Source:   domain.SourceService,
			Type:     "WORK_ORDER",
			Title:    "60,000 mile service",
			IssuedAt: issued,
			URL:      "http://localhost:9101/service/v1/attachments/WO-77/raw",
		}},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK, LatencyMs: 42, DocumentCount: 0},
			{Name: domain.SourceService, Status: domain.SourceStatusTimeout, ErrorCode: domain.CodeUpstreamTimeout, LatencyMs: 2001},
		},
		Partial: true,
	})
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, key := range []string{`"issued_at"`, `"served_from_cache"`, `"request_id"`, `"latency_ms"`, `"document_count"`} {
		if !strings.Contains(body, key) {
			t.Fatalf("missing snake_case key %s in %s", key, body)
		}
	}
	for _, leak := range []string{"IssuedAt", "ServedFromCache", "RequestID", "LatencyMs"} {
		if strings.Contains(body, leak) {
			t.Fatalf("PascalCase leaked: %s", body)
		}
	}
	if got.Documents[0].IssuedAt != "2025-01-04T09:30:00Z" {
		t.Fatalf("issued_at = %q", got.Documents[0].IssuedAt)
	}
	if got.Sources[1].Status != "UNAVAILABLE" || got.Sources[1].Error == nil || got.Sources[1].Error.Code != domain.CodeUpstreamTimeout {
		t.Fatalf("failed source wire = %+v", got.Sources[1])
	}
	if got.Sources[1].Error.Message != "service fetch exceeded per-source budget" {
		t.Fatalf("error message = %q", got.Sources[1].Error.Message)
	}
	if strings.Contains(strings.ToLower(got.Sources[1].Error.Message), "localhost") {
		t.Fatalf("error message leaked host: %s", got.Sources[1].Error.Message)
	}
}

func TestFromAggregateEmptyDocumentsIsArray(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(FromAggregate("3N1AB7AP1D", "req-2", domain.AggregateResult{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"documents":[]`) {
		t.Fatalf("FR6 empty list must be []: %s", raw)
	}
	if strings.Contains(string(raw), `"documents":null`) {
		t.Fatalf("documents was null: %s", raw)
	}
}
