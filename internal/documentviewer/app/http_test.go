package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	docsapi "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/api"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
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
