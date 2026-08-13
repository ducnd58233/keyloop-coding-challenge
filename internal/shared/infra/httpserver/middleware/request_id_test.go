package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDClipsLongInboundValue(t *testing.T) {
	t.Parallel()
	h := RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", strings.Repeat("r", 200))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	got := rec.Header().Get("X-Request-Id")
	if len([]rune(got)) != maxRequestIDLen {
		t.Fatalf("len = %d, want %d", len([]rune(got)), maxRequestIDLen)
	}
}
