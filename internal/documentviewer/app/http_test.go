package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	docsapi "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/api"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/openapidocs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

type stubDocsQuery struct {
	err error
}

func (s stubDocsQuery) Documents(context.Context, string) (domain.AggregateResult, error) {
	return domain.AggregateResult{}, s.err
}

func TestHealthz(t *testing.T) {
	h := mountHTTP(httpDeps{log: observability.Discard()})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}

func TestDocumentsRouteMounted(t *testing.T) {
	h := mountHTTP(httpDeps{
		log:  observability.Discard(),
		docs: docsapi.New(stubDocsQuery{err: domain.ErrInvalidVIN}, nil, observability.Discard()),
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res, err := srv.Client().Get(srv.URL + "/api/v1/vehicles/1HGCM8263/documents")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
}

func TestOpenAPIContractMounted(t *testing.T) {
	h := mountHTTP(httpDeps{log: observability.Discard()})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	res, err := srv.Client().Get(srv.URL + openapidocs.PathJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("openapi.json status = %d", res.StatusCode)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, "/vehicles/{vin}/documents") {
		t.Fatalf("contract missing documents path: %s", body)
	}
	if !strings.Contains(body, `"message"`) || strings.Contains(body, "dto.ErrorBody") {
		t.Fatalf("contract still has old error envelope: %s", body)
	}

	ui, err := srv.Client().Get(srv.URL + openapidocs.PathUI)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ui.Body.Close() }()
	if ui.StatusCode != http.StatusOK {
		t.Fatalf("docs status = %d", ui.StatusCode)
	}
	html, err := io.ReadAll(ui.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)
	if !strings.Contains(page, openapidocs.PathJSON) {
		t.Fatalf("docs missing spec url: %s", page)
	}
	for _, vin := range []string{testutil.TestVIN, mockseed.WithNone} {
		if strings.Contains(page, vin) {
			t.Fatalf("docs HTML leaked VIN %s", vin)
		}
	}
}
