//go:build integration

package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers pgx5://
	"github.com/golang-migrate/migrate/v4/source/iofs"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
	"github.com/ducnd58233/unified-document-viewer/migrations"
)

const (
	postgresImage    = "postgres:18-alpine"
	testDBName       = "itest"
	testDBUser       = "itest"
	testDBPassword   = "itest"
	testPoolMaxConns = 4
	migrateScheme    = "pgx5://"
	postgresScheme   = "postgres://"
	initTimeout      = 90 * time.Second
)

var (
	startOnce sync.Once
	shared    *postgres.Pool
	errStart  error
)

// OpenPool returns a pool against an isolated Testcontainers Postgres 18
// instance with migrations applied. It never reads configs.DATABASE_URL or the
// compose volume (K5 demo DB stays out of adapter tests).
func OpenPool(t *testing.T) *postgres.Pool {
	t.Helper()
	startOnce.Do(func() {
		errStart = startIsolated(context.Background())
	})
	if errStart != nil {
		t.Fatalf("isolated postgres: %v", errStart)
	}
	return shared
}

func startIsolated(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()

	ctr, err := tcpostgres.Run(ctx,
		postgresImage,
		tcpostgres.WithDatabase(testDBName),
		tcpostgres.WithUsername(testDBUser),
		tcpostgres.WithPassword(testDBPassword),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return fmt.Errorf("testutil: start postgres container: %w", err)
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("testutil: connection string: %w", err)
	}

	src, err := iofs.New(migrations.SQL, ".")
	if err != nil {
		return fmt.Errorf("testutil: migrate source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, strings.Replace(connStr, postgresScheme, migrateScheme, 1))
	if err != nil {
		return fmt.Errorf("testutil: migrate open: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("testutil: migrate up: %w", err)
	}

	p, err := postgres.Open(ctx, connStr, testPoolMaxConns)
	if err != nil {
		return fmt.Errorf("testutil: open pool: %w", err)
	}
	shared = p
	return nil
}

// VINFor is unique per test name so parallel integration tests do not share rows.
func VINFor(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(t.Name()))
	return strings.ToUpper(hex.EncodeToString(sum[:]))[:10]
}
