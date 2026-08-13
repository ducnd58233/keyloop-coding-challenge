// Package httpclient is outbound HTTP transport shared by binaries. It does
// not know Sales or Service shapes; FR5 normalisers stay in the documents
// module.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/randutil"
)

const maxBody = 1 << 20

// Client issues JSON GETs with at most one retry on transport errors and 5xx.
// Timeouts never retry (NFR2 budget is already spent).
type Client struct {
	http  *http.Client
	sleep func(context.Context, time.Duration) error
	retry bool
}

// Option configures a Client. Defaults match the aggregator: ctx deadlines,
// one retry, short jitter.
type Option func(*Client)

// WithHTTPClient replaces the default transport.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		if c != nil {
			cl.http = c
		}
	}
}

// WithSleep replaces jitter wait (tests pass a no-op).
func WithSleep(fn func(context.Context, time.Duration) error) Option {
	return func(cl *Client) {
		if fn != nil {
			cl.sleep = fn
		}
	}
}

// WithoutRetry disables the single 5xx/transport retry.
func WithoutRetry() Option {
	return func(cl *Client) { cl.retry = false }
}

// New uses a small idle pool per host. Request timeouts live on ctx, not here.
func New(opts ...Option) *Client {
	c := &Client{
		http: &http.Client{
			Transport: &http.Transport{MaxIdleConnsPerHost: 8},
		},
		sleep: sleep,
		retry: true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type statusError struct{ code int }

func (e statusError) Error() string {
	return fmt.Sprintf("upstream status %d", e.code)
}

// DoJSON never puts the request URL, hostname, or response body into the error.
func (c *Client) DoJSON(ctx context.Context, req *http.Request, dest any) error {
	if c == nil || c.http == nil {
		return errors.New("upstream fetch failed")
	}
	if id := httpserver.RequestIDFrom(ctx); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	err := c.doOnce(req, dest)
	if err == nil || !c.retry || !retryable(err) {
		return err
	}
	if err := c.sleep(ctx, jitter()); err != nil {
		return err
	}
	return c.doOnce(req, dest)
}

func (c *Client) doOnce(req *http.Request, dest any) error {
	res, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return err
		}
		return errors.New("upstream fetch failed")
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBody))
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		return statusError{code: res.StatusCode}
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(dest); err != nil {
		return errors.New("upstream decode failed")
	}
	return nil
}

func retryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var st statusError
	if errors.As(err, &st) {
		return st.code >= 500
	}
	return true
}

func jitter() time.Duration {
	d, err := randutil.DurationBetween(10*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		return 10 * time.Millisecond
	}
	return d
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
