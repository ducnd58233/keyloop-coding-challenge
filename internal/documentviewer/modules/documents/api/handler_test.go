package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/dto"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver/middleware"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

const testVIN = testutil.TestVIN

type stubQuery struct {
	result domain.AggregateResult
	err    error
}

func (s stubQuery) Documents(context.Context, string) (domain.AggregateResult, error) {
	return s.result, s.err
}

type recLog struct {
	vin, actor, reqID string
	err               error
	n                 int
}

func (r *recLog) Record(_ context.Context, vin, actorID, requestID string, _ domain.AggregateResult, err error, _ time.Duration) {
	r.n++
	r.vin, r.actor, r.reqID, r.err = vin, actorID, requestID, err
}

func serve(h *Handler, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	chain := middleware.Chain(mux, middleware.RequestID)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)
	return rec
}

func TestListInvalidVINIs400AndAudited(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{err: domain.ErrInvalidVIN}, audit, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/1HGCM8263/documents", nil)
	req.Header.Set("X-Actor-Id", "advisor-1")
	rec := serve(h, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != domain.CodeInvalidVIN {
		t.Fatalf("code = %q", body.Error.Code)
	}
	if audit.n != 1 || audit.vin != "1HGCM8263" || audit.actor != "advisor-1" || !errors.Is(audit.err, domain.ErrInvalidVIN) {
		t.Fatalf("audit = %+v", audit)
	}
}

func TestListUnknownVINIs200EmptyArray(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{
		Documents: []domain.Document{},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK},
			{Name: domain.SourceService, Status: domain.SourceStatusOK},
		},
	}}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/3N1AB7AP1D/documents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body dto.DocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Partial || len(body.Documents) != 0 || len(body.Sources) != 2 {
		t.Fatalf("FR6/FR7 body = %+v", body)
	}
	if audit.n != 1 {
		t.Fatalf("FR8 audit missing: %+v", audit)
	}
}

func TestListBothOKIs200NotPartial(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{
		Documents: []domain.Document{
			{ID: "sales:1", Source: domain.SourceSales, Type: "INVOICE", IssuedAt: time.Unix(0, 0).UTC()},
			{ID: "service:2", Source: domain.SourceService, Type: "WORK_ORDER", IssuedAt: time.Unix(1, 0).UTC()},
		},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK, DocumentCount: 1},
			{Name: domain.SourceService, Status: domain.SourceStatusOK, DocumentCount: 1},
		},
	}}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body dto.DocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Partial || len(body.Documents) != 2 || len(body.Sources) != 2 {
		t.Fatalf("body = %+v", body)
	}
	if audit.n != 1 {
		t.Fatalf("FR8 audit missing: %+v", audit)
	}
}

func TestListUnexpectedErrorIs503AllSourcesUnavailable(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{err: errors.New("boom")}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.CodeAllSourcesUnavailable) || strings.Contains(rec.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if audit.n != 1 {
		t.Fatalf("FR8 audit missing: %+v", audit)
	}
}

func TestListAllSourcesDownIs503(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{
		result: domain.AggregateResult{Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusError, ErrorCode: domain.CodeUpstreamError},
			{Name: domain.SourceService, Status: domain.SourceStatusError, ErrorCode: domain.CodeUpstreamError},
		}},
		err: domain.ErrAllSourcesUnavailable,
	}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != domain.CodeAllSourcesUnavailable {
		t.Fatalf("code = %q", body.Error.Code)
	}
	if audit.n != 1 || !errors.Is(audit.err, domain.ErrAllSourcesUnavailable) {
		t.Fatalf("FR8 audit missing: %+v", audit)
	}
}

func TestListPartialIs200WithSources(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{
		Documents: []domain.Document{{
			ID:       "sales:INV-1",
			Source:   domain.SourceSales,
			Type:     "INVOICE",
			Title:    "Inv",
			IssuedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			URL:      "https://sales.example/1",
		}},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK, DocumentCount: 1, LatencyMs: 10},
			{Name: domain.SourceService, Status: domain.SourceStatusTimeout, ErrorCode: domain.CodeUpstreamTimeout, LatencyMs: 2001},
		},
		Partial: true,
	}}, audit, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil)
	req.Header.Set("X-Request-Id", "fixed-req")
	rec := serve(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body dto.DocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Partial || len(body.Documents) != 1 || len(body.Sources) != 2 || body.RequestID != "fixed-req" {
		t.Fatalf("FR7 body = %+v", body)
	}
	if rec.Header().Get("X-Request-Id") != "fixed-req" {
		t.Fatalf("X-Request-Id = %q", rec.Header().Get("X-Request-Id"))
	}
	if audit.n != 1 {
		t.Fatalf("FR8 audit missing: %+v", audit)
	}
}

func TestListDoesNotLogFullVIN(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	h := New(stubQuery{err: domain.ErrInvalidVIN}, &recLog{}, log)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil)
	_ = serve(h, req)
	out := buf.String()
	if strings.Contains(out, testVIN) || strings.Contains(out, "/api/v1/vehicles/"+testVIN) {
		t.Fatalf("full VIN leaked in logs: %s", out)
	}
	if !strings.Contains(out, "vin_suffix=2633") || !strings.Contains(out, routeTemplate) {
		t.Fatalf("want route template + suffix, got %s", out)
	}
}

func TestListPropagatesRequestIDToAudit(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{}}, audit, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil)
	req.Header.Set("X-Request-Id", "abc-123")
	_ = serve(h, req)
	if audit.reqID != "abc-123" {
		t.Fatalf("audit request id = %q", audit.reqID)
	}
}

func TestClipHeaderBoundsActor(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 200)
	if got := clipHeader(long); utf8Count(got) != maxHeaderLen {
		t.Fatalf("len = %d", utf8Count(got))
	}
}

func utf8Count(s string) int {
	return len([]rune(s))
}

func TestListFreshCacheIs200ServedFromCache(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{
		Documents: []domain.Document{{ID: "sales:1", Source: domain.SourceSales, Type: "INVOICE", IssuedAt: time.Unix(0, 0).UTC()}},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK},
			{Name: domain.SourceService, Status: domain.SourceStatusOK},
		},
		ServedFromCache: true,
	}}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body dto.DocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.ServedFromCache || len(body.Sources) != 2 || audit.n != 1 {
		t.Fatalf("body=%+v audit=%+v", body, audit)
	}
}

func TestListStaleIs200(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{
		Documents: []domain.Document{{ID: "sales:1", Source: domain.SourceSales, Type: "INVOICE", IssuedAt: time.Unix(0, 0).UTC()}},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK},
			{Name: domain.SourceService, Status: domain.SourceStatusOK},
		},
		Stale: true,
	}}, audit, nil)
	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body dto.DocumentsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Stale || len(body.Sources) != 2 || audit.n != 1 {
		t.Fatalf("FR10 body=%+v audit=%+v", body, audit)
	}
}

type blockingQuery struct{}

func (blockingQuery) Documents(ctx context.Context, _ string) (domain.AggregateResult, error) {
	<-ctx.Done()
	return domain.AggregateResult{}, ctx.Err()
}

func TestListRequestTimeoutIs503(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(blockingQuery{}, audit, nil)
	mux := http.NewServeMux()
	h.Register(mux)
	chain := middleware.Chain(mux, middleware.RequestID, middleware.Timeout(30*time.Millisecond))
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.CodeAllSourcesUnavailable) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if audit.n != 1 || !errors.Is(audit.err, context.DeadlineExceeded) && !errors.Is(audit.err, context.Canceled) {
		t.Fatalf("FR8 timeout audit = %+v", audit)
	}
}

func TestListClipsLongRequestIDOnAudit(t *testing.T) {
	t.Parallel()
	audit := &recLog{}
	h := New(stubQuery{result: domain.AggregateResult{}}, audit, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles/"+testVIN+"/documents", nil)
	req.Header.Set("X-Request-Id", strings.Repeat("r", 200))
	_ = serve(h, req)
	if utf8Count(audit.reqID) != maxHeaderLen {
		t.Fatalf("audit request id len = %d", utf8Count(audit.reqID))
	}
}
