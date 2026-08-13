package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutCancelsSlowHandler(t *testing.T) {
	t.Parallel()
	h := Timeout(20 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		if r.Context().Err() != context.DeadlineExceeded {
			t.Errorf("ctx err = %v, want DeadlineExceeded", r.Context().Err())
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestTimeoutZeroIsNoOp(t *testing.T) {
	t.Parallel()
	h := Timeout(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); ok {
			t.Fatal("zero timeout must not set a deadline")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}
