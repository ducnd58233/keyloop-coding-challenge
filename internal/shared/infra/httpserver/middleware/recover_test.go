package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func TestRecoverMapsPanicToInternalError(t *testing.T) {
	h := Recover(observability.NewLogger("error", io.Discard))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	body := rec.Body.String()
	lower := strings.ToLower(body)
	if strings.Contains(lower, "goroutine") || strings.Contains(lower, "panic") || strings.Contains(body, ".go:") {
		t.Fatalf("response leaked internals: %s", body)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("code = %q, want INTERNAL_ERROR", payload.Error.Code)
	}
}
