// Command service is the Service upstream mock (A5).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	app "github.com/ducnd58233/unified-document-viewer/internal/service/app"
)

func main() {
	addr := flag.String("addr", ":9101", "listen address")
	baseURL := flag.String("base-url", "http://localhost:9101", "absolute base used in file.uri")
	down := flag.Bool("down", false, "simulate a full outage: every route returns 503")
	deterministic := flag.Bool("deterministic", false, "stable catalog only; no random faults or generated VINs")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, app.RunOptions{
		Addr:          *addr,
		BaseURL:       *baseURL,
		Down:          *down,
		Deterministic: *deterministic,
	}); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "service mock stopped: %v\n", err)
		os.Exit(1)
	}
}
