package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenInvalidURLOmitsSecret(t *testing.T) {
	t.Parallel()
	_, err := Open(context.Background(), "://not-a-url", 2)
	if err == nil {
		t.Fatal("Open() error = nil, want invalid url")
	}
	msg := strings.ToLower(err.Error())
	for _, leak := range []string{"password", "secret", "://not-a-url"} {
		if strings.Contains(msg, leak) {
			t.Fatalf("error leaked connection detail %q: %v", leak, err)
		}
	}
}

func TestOpenUnreachableOmitsSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Open(ctx, "postgres://viewer:s3cret@127.0.0.1:1/viewer?sslmode=disable", 2)
	if err == nil {
		t.Fatal("Open() error = nil, want connect/ping failure")
	}
	msg := strings.ToLower(err.Error())
	for _, leak := range []string{"s3cret", "password", "127.0.0.1", "viewer", "postgres://", "host="} {
		if strings.Contains(msg, leak) {
			t.Fatalf("error leaked connection detail %q: %v", leak, err)
		}
	}
}

func TestOpenNilPingAfterClose(t *testing.T) {
	t.Parallel()
	var p *Pool
	if err := p.Ping(context.Background()); err == nil {
		t.Fatal("nil pool Ping() error = nil")
	}
}
