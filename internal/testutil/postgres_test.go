//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
)

func TestOpenPoolIgnoresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://viewer:viewer@127.0.0.1:1/viewer?sslmode=disable")
	pool := OpenPool(t)
	ctx := context.Background()
	var one int
	if err := postgres.QuerierFrom(ctx, pool).QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil {
		t.Fatalf("query isolated pool: %v", err)
	}
	if one != 1 {
		t.Fatalf("SELECT 1 = %d", one)
	}
}
