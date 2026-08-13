package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func TestHealthEndpoints(t *testing.T) {
	h := mountHTTP(httpDeps{log: observability.NewLogger("error", io.Discard)})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			res, err := srv.Client().Get(srv.URL + path)
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
		})
	}
}
