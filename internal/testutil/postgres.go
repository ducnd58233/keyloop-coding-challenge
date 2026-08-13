//go:build integration

package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/configs"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
)

// OpenPool fails loudly if Postgres is not up (make infra-up migrate-up).
func OpenPool(t *testing.T) *postgres.Pool {
	t.Helper()
	cfg, err := configs.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	p, err := postgres.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
	if err != nil {
		t.Fatalf("open test pool: %v (run make infra-up migrate-up)", err)
	}
	t.Cleanup(p.Close)
	return p
}

// VINFor is unique per test name so parallel integration tests do not share rows.
func VINFor(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(t.Name()))
	return strings.ToUpper(hex.EncodeToString(sum[:]))[:10]
}
