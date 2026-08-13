package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func TestDoJSONOKAndRequestID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-Id") != "req-1" {
			t.Fatalf("X-Request-Id = %q", r.Header.Get("X-Request-Id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(httpserver.WithRequestID(context.Background(), "req-1"), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := New().DoJSON(req.Context(), req, &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != "yes" {
		t.Fatalf("got = %#v", got)
	}
}

func TestDoJSONRetries5xxOnce(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"n": 2})
	}))
	t.Cleanup(srv.Close)

	c := New(WithSleep(func(context.Context, time.Duration) error { return nil }))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]int
	if err := c.DoJSON(req.Context(), req, &got); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 || got["n"] != 2 {
		t.Fatalf("hits=%d got=%#v", hits.Load(), got)
	}
}

func TestDoJSONDoesNotRetry4xx(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c := New(WithSleep(func(context.Context, time.Duration) error { return nil }))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(req.Context(), req, new(map[string]any))
	if err == nil {
		t.Fatal("want error")
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want 1", hits.Load())
	}
	if strings.Contains(err.Error(), "localhost") || strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("error leaked host: %v", err)
	}
}

func TestDoJSONDoesNotRetryTimeout(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	c := New(WithSleep(func(context.Context, time.Duration) error {
		t.Fatal("timeout must not sleep for retry")
		return nil
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = c.DoJSON(ctx, req, new(map[string]any))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if strings.Contains(err.Error(), "localhost") || strings.Contains(err.Error(), srv.URL) || strings.Contains(err.Error(), "http") {
		t.Fatalf("timeout error leaked host: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d, want 1", hits.Load())
	}
}

func TestDoJSONRetriesTransportOnce(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	c := New(
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if hits.Add(1) == 1 {
				return nil, errors.New("connection reset")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		})}),
		WithSleep(func(context.Context, time.Duration) error { return nil }),
	)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://upstream.example/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]bool
	if err := c.DoJSON(req.Context(), req, &got); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 || !got["ok"] {
		t.Fatalf("hits=%d got=%v", hits.Load(), got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDoJSONDecodeErrorOmitsBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vin":"`+testutil.TestVIN+`"`)
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = New().DoJSON(req.Context(), req, new(map[string]any))
	if err == nil {
		t.Fatal("want decode error")
	}
	if strings.Contains(err.Error(), testutil.TestVIN) || strings.Contains(err.Error(), "localhost") {
		t.Fatalf("error leaked vin or host: %v", err)
	}
}
