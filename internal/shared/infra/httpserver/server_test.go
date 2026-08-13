package httpserver

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func TestServeReturnsBindError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	t.Cleanup(func() { _ = ln.Close() })

	err = Serve(context.Background(), addr, http.NewServeMux(), observability.Discard())
	if err == nil {
		t.Fatal("expected listen error when address is already bound")
	}
}

func TestServeShutsDownOnCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, addr, http.NewServeMux(), observability.Discard())
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		c, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = c.Close()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server never accepted connections: %v", dialErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve = %v, want nil after cancel", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
}
