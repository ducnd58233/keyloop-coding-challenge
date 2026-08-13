//go:build integration

package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func fullResult() domain.AggregateResult {
	issued := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	return domain.AggregateResult{
		Documents: []domain.Document{{
			ID:       "sales:1",
			Source:   domain.SourceSales,
			Type:     "invoice",
			Title:    "Inv",
			IssuedAt: issued,
			URL:      "https://sales.example/1",
		}},
		Sources: []domain.SourceReport{
			{Name: domain.SourceSales, Status: domain.SourceStatusOK},
			{Name: domain.SourceService, Status: domain.SourceStatusOK},
		},
	}
}

func TestCacheFreshThenStaleWithFakeClock(t *testing.T) {
	pool := testutil.OpenPool(t)
	start := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := common.NewFixedClock(start)
	repo := NewCacheRepository(pool, clock)
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	if err := repo.Store(ctx, vin, fullResult(), time.Minute); err != nil {
		t.Fatal(err)
	}
	hit, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok {
		t.Fatalf("fresh lookup ok=%v err=%v", ok, err)
	}
	if hit.Stale || len(hit.Result.Documents) != 1 || hit.Result.Documents[0].ID != "sales:1" {
		t.Fatalf("fresh hit = %+v", hit)
	}
	want := fullResult()
	gotDoc, wantDoc := hit.Result.Documents[0], want.Documents[0]
	if gotDoc.URL != wantDoc.URL || gotDoc.Source != wantDoc.Source || gotDoc.Type != wantDoc.Type ||
		gotDoc.Title != wantDoc.Title || !gotDoc.IssuedAt.Equal(wantDoc.IssuedAt) {
		t.Fatalf("A4 round-trip lost document fields: %+v", gotDoc)
	}
	if len(hit.Result.Sources) != 2 ||
		hit.Result.Sources[0] != want.Sources[0] || hit.Result.Sources[1] != want.Sources[1] {
		t.Fatalf("FR7 round-trip lost sources[]: %+v", hit.Result.Sources)
	}
	if hit.Result.Partial || hit.Result.Stale || hit.Result.ServedFromCache {
		t.Fatalf("stored flags leaked into payload: %+v", hit.Result)
	}

	clock.Advance(61 * time.Second)
	stale, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok {
		t.Fatalf("stale lookup ok=%v err=%v", ok, err)
	}
	if !stale.Stale {
		t.Fatal("FR10: expired row must be stale, not deleted")
	}
	if stale.Result.Documents[0].ID != "sales:1" {
		t.Fatalf("stale payload lost documents: %+v", stale.Result)
	}
}

func TestCacheStoreUpsertReplacesPayloadAndTTL(t *testing.T) {
	pool := testutil.OpenPool(t)
	start := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := common.NewFixedClock(start)
	repo := NewCacheRepository(pool, clock)
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	first := fullResult()
	if err := repo.Store(ctx, vin, first, time.Minute); err != nil {
		t.Fatal(err)
	}
	second := fullResult()
	second.Documents[0].ID = "sales:2"
	second.Documents[0].URL = "https://sales.example/2"
	if err := repo.Store(ctx, vin, second, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	hit, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok || hit.Result.Documents[0].ID != "sales:2" {
		t.Fatalf("upsert lookup = ok=%v err=%v hit=%+v", ok, err, hit)
	}
	clock.Advance(61 * time.Second)
	stillFresh, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok || stillFresh.Stale {
		t.Fatalf("new TTL should still be fresh at +61s: ok=%v stale=%v err=%v", ok, stillFresh.Stale, err)
	}
	clock.Advance(60 * time.Second)
	stale, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok || !stale.Stale {
		t.Fatalf("want stale after +121s: ok=%v stale=%v err=%v", ok, stale.Stale, err)
	}
}

func TestCacheStoreRejectsPartial(t *testing.T) {
	pool := testutil.OpenPool(t)
	repo := NewCacheRepository(pool, common.NewFixedClock(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	if err := repo.Store(ctx, vin, fullResult(), time.Minute); err != nil {
		t.Fatal(err)
	}
	partial := fullResult()
	partial.Partial = true
	if err := repo.Store(ctx, vin, partial, time.Minute); !errors.Is(err, ErrPartialNotCached) {
		t.Fatalf("Store(partial) err = %v, want ErrPartialNotCached", err)
	}
	hit, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok || hit.Result.Partial || len(hit.Result.Documents) != 1 {
		t.Fatalf("NFR6: partial must not poison a full row, ok=%v err=%v hit=%+v", ok, err, hit)
	}
}

func TestCacheStoreJoinsUnitOfWorkRollback(t *testing.T) {
	pool := testutil.OpenPool(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := NewCacheRepository(pool, common.NewFixedClock(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	err := uow.Within(ctx, func(ctx context.Context) error {
		if err := repo.Store(ctx, vin, fullResult(), time.Minute); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("Within error = nil, want rollback")
	}
	_, ok, err := repo.Lookup(ctx, vin)
	if err != nil || ok {
		t.Fatalf("R4: rolled-back store visible, ok=%v err=%v", ok, err)
	}
}

func TestCacheUnitOfWorkRollbackAfterCancel(t *testing.T) {
	pool := testutil.OpenPool(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := NewCacheRepository(pool, common.NewFixedClock(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	reqCtx, cancel := context.WithCancel(ctx)
	err := uow.Within(reqCtx, func(txCtx context.Context) error {
		if err := repo.Store(txCtx, vin, fullResult(), time.Minute); err != nil {
			return err
		}
		cancel()
		return errors.New("force rollback after cancel")
	})
	if err == nil {
		t.Fatal("Within error = nil, want rollback")
	}
	if pingErr := pool.Ping(ctx); pingErr != nil {
		t.Fatalf("pool ping after cancelled Within: %v", pingErr)
	}
	_, ok, lookupErr := repo.Lookup(ctx, vin)
	if lookupErr != nil || ok {
		t.Fatalf("cancelled Within leaked row ok=%v err=%v", ok, lookupErr)
	}
}

func TestCacheStoreUnitOfWorkCommit(t *testing.T) {
	pool := testutil.OpenPool(t)
	uow := postgres.NewUnitOfWork(pool)
	repo := NewCacheRepository(pool, common.NewFixedClock(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)))
	vin := testutil.VINFor(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = postgres.QuerierFrom(ctx, pool).Exec(ctx, `DELETE FROM document_cache WHERE vin = $1`, vin)
	})

	if err := uow.Within(ctx, func(ctx context.Context) error {
		return repo.Store(ctx, vin, fullResult(), time.Minute)
	}); err != nil {
		t.Fatal(err)
	}
	hit, ok, err := repo.Lookup(ctx, vin)
	if err != nil || !ok || len(hit.Result.Documents) != 1 {
		t.Fatalf("committed lookup ok=%v err=%v hit=%+v", ok, err, hit)
	}
}

func TestCacheLookupErrorOnClosedPool(t *testing.T) {
	pool := testutil.OpenPool(t)
	repo := NewCacheRepository(pool, common.SystemClock{})
	pool.Close()
	_, _, err := repo.Lookup(context.Background(), testutil.VINFor(t))
	if err == nil {
		t.Fatal("Lookup on closed pool err = nil, want error (NFR7 caller fails open)")
	}
}
