//go:build integration

package persistence

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app/usecases"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func TestAuditRecordIsAppendOnly(t *testing.T) {
	pool := testutil.OpenPool(t)
	repo := NewAuditRepository(pool)
	vin := testutil.VINFor(t)
	hash, suffix := common.HashVIN("unit-test-salt", vin)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM search_audit WHERE vin_hash = $1`, hash)
	})

	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	for i, outcome := range []domain.Outcome{domain.OutcomeInvalidVIN, domain.OutcomeUnavailable} {
		err := repo.Record(ctx, domain.AccessEvent{
			VINHash:       hash,
			VINSuffix:     suffix,
			ActorID:       "tech-1",
			RequestID:     fmt.Sprintf("req-%d", i),
			TraceID:       "tr-1",
			Outcome:       outcome,
			SourcesOK:     0,
			SourcesFailed: 2,
			Latency:       time.Duration(i+1) * 10 * time.Millisecond,
			RequestedAt:   now.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	rows, err := postgres.QuerierFrom(ctx, pool).Query(ctx, `
		SELECT outcome, vin_suffix, latency_ms
		FROM search_audit
		WHERE vin_hash = $1
		ORDER BY requested_at
	`, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var outcomes []string
	for rows.Next() {
		var outcome, gotSuffix string
		var latencyMS int
		if err := rows.Scan(&outcome, &gotSuffix, &latencyMS); err != nil {
			t.Fatal(err)
		}
		if gotSuffix != suffix {
			t.Fatalf("vin_suffix = %q, want %q", gotSuffix, suffix)
		}
		outcomes = append(outcomes, outcome)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 2 || outcomes[0] != string(domain.OutcomeInvalidVIN) || outcomes[1] != string(domain.OutcomeUnavailable) {
		t.Fatalf("NFR8: outcomes = %v, want two appended rows", outcomes)
	}
}

func TestRecordAccessExecuteDoesNotPersistFullVIN(t *testing.T) {
	pool := testutil.OpenPool(t)
	repo := NewAuditRepository(pool)
	const vin = "1HGCM82633"
	const salt = "unit-test-salt"
	wantHash, wantSuffix := common.HashVIN(salt, vin)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM search_audit WHERE vin_hash = $1`, wantHash)
	})

	u := usecases.NewRecordAccess(repo, salt, common.NewFixedClock(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	if err := u.Execute(ctx, usecases.AccessInput{
		VIN:     vin,
		Outcome: domain.OutcomeInvalidVIN,
		Latency: 5 * time.Millisecond,
	}); err != nil {
		t.Fatal(err)
	}

	var hash, suffix string
	err := postgres.QuerierFrom(ctx, pool).QueryRow(ctx, `
		SELECT vin_hash, vin_suffix FROM search_audit WHERE vin_hash = $1
	`, wantHash).Scan(&hash, &suffix)
	if err != nil {
		t.Fatal(err)
	}
	if hash != wantHash || suffix != wantSuffix || hash == vin || suffix == vin {
		t.Fatalf("persisted hash/suffix = %q %q", hash, suffix)
	}
}

func TestAuditRecordJoinsUnitOfWorkRollback(t *testing.T) {
	pool := testutil.OpenPool(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := NewAuditRepository(pool)
	hash, suffix := common.HashVIN("unit-test-salt", testutil.VINFor(t))
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM search_audit WHERE vin_hash = $1`, hash)
	})

	err := uow.Within(ctx, func(ctx context.Context) error {
		if err := repo.Record(ctx, domain.AccessEvent{
			VINHash:     hash,
			VINSuffix:   suffix,
			Outcome:     domain.OutcomeUnavailable,
			RequestedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("Within error = nil, want rollback")
	}
	var count int
	if err := postgres.QuerierFrom(ctx, pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM search_audit WHERE vin_hash = $1
	`, hash).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rolled-back audit still visible, count=%d", count)
	}
}

func TestAuditRecordDoesNotStoreFullVIN(t *testing.T) {
	pool := testutil.OpenPool(t)
	repo := NewAuditRepository(pool)
	vin := testutil.VINFor(t)
	hash, suffix := common.HashVIN("unit-test-salt", vin)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM search_audit WHERE vin_hash = $1`, hash)
	})

	if err := repo.Record(ctx, domain.AccessEvent{
		VINHash:     hash,
		VINSuffix:   suffix,
		Outcome:     domain.OutcomeOK,
		RequestedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	var count int
	err := postgres.QuerierFrom(ctx, pool).QueryRow(ctx, `
		SELECT COUNT(*) FROM search_audit
		WHERE vin_hash = $1 AND (
			vin_hash = $2 OR vin_suffix = $2 OR actor_id = $2 OR request_id = $2 OR trace_id = $2
		)
	`, hash, vin).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("full VIN appeared in audit columns, count=%d", count)
	}
}
