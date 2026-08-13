package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func TestRecoverMapsPanicToInternalError(t *testing.T) {
	h := Recover(observability.Discard())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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

func TestRecoverLogOmitsPanicVIN(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	h := Recover(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(testutil.TestVIN)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	out := buf.String()
	if strings.Contains(out, testutil.TestVIN) {
		t.Fatalf("panic value leaked into log: %s", out)
	}
	if !strings.Contains(out, "panic_type") {
		t.Fatalf("want panic_type, got %s", out)
	}
}
