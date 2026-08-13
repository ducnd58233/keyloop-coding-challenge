package usecases

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	appmocks "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app/mocks"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

const testVIN = "1HGCM82633"

func salesDoc(id string) domain.Document {
	return domain.Document{
		ID:       domain.NamespacedID(domain.SourceSales, id),
		Source:   domain.SourceSales,
		Type:     "invoice",
		Title:    id,
		IssuedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		URL:      "https://sales.example/" + id,
	}
}

func serviceDoc(id string) domain.Document {
	return domain.Document{
		ID:       domain.NamespacedID(domain.SourceService, id),
		Source:   domain.SourceService,
		Type:     "WORK_ORDER",
		Title:    id,
		IssuedAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		URL:      "https://service.example/" + id,
	}
}

func missCache(ctrl *gomock.Controller) *appmocks.MockDocumentCache {
	c := appmocks.NewMockDocumentCache(ctrl)
	c.EXPECT().Lookup(gomock.Any(), gomock.Any()).Return(domain.CachedResult{}, false, nil).AnyTimes()
	return c
}

func stubSource(ctrl *gomock.Controller, name domain.SourceName, docs []domain.Document, err error) *appmocks.MockDocumentSource {
	s := appmocks.NewMockDocumentSource(ctrl)
	s.EXPECT().Name().Return(name).AnyTimes()
	s.EXPECT().Fetch(gomock.Any(), testVIN).Return(docs, err)
	return s
}

func delayedSource(
	ctrl *gomock.Controller,
	name domain.SourceName,
	delay time.Duration,
	docs []domain.Document,
	err error,
	cancelled *atomic.Bool,
) *appmocks.MockDocumentSource {
	s := appmocks.NewMockDocumentSource(ctrl)
	s.EXPECT().Name().Return(name).AnyTimes()
	s.EXPECT().Fetch(gomock.Any(), testVIN).DoAndReturn(func(ctx context.Context, _ string) ([]domain.Document, error) {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			if cancelled != nil {
				cancelled.Store(true)
			}
			return nil, ctx.Err()
		case <-timer.C:
			if e := ctx.Err(); e != nil {
				if cancelled != nil {
					cancelled.Store(true)
				}
				return nil, e
			}
			return docs, err
		}
	})
	return s
}

func sourceReport(t *testing.T, reports []domain.SourceReport, name domain.SourceName) domain.SourceReport {
	t.Helper()
	for _, r := range reports {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("missing FR7 report for %s in %+v", name, reports)
	return domain.SourceReport{}
}

func newAgg(ctrl *gomock.Controller, allowStore bool, sources ...app.DocumentSource) *Aggregate {
	cache := missCache(ctrl)
	if allowStore {
		cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	} else {
		cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	}
	return New(Options{
		Sources:          sources,
		DocumentCache:    cache,
		PerSourceTimeout: 2 * time.Second,
		AggregateTimeout: 2500 * time.Millisecond,
		CacheTTL:         time.Minute,
	})
}

func TestDocumentsRejectsInvalidVIN(t *testing.T) {
	t.Parallel()
	a := New(Options{})
	for _, vin := range []string{"1HGCM8263", "1HGCM826331"} {
		if _, err := a.Documents(context.Background(), vin); !errors.Is(err, domain.ErrInvalidVIN) {
			t.Fatalf("vin %q err = %v, want ErrInvalidVIN", vin, err)
		}
	}
}

func TestDocumentsBothOK(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	cache := missCache(ctrl)
	var stored *domain.AggregateResult
	var storeTTL time.Duration
	cache.EXPECT().Store(gomock.Any(), testVIN, gomock.Any(), time.Minute).
		DoAndReturn(func(_ context.Context, _ string, r domain.AggregateResult, ttl time.Duration) error {
			cp := r
			stored = &cp
			storeTTL = ttl
			return nil
		})
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			stubSource(ctrl, domain.SourceService, []domain.Document{serviceDoc("2")}, nil),
		},
		DocumentCache:    cache,
		PerSourceTimeout: time.Second,
		CacheTTL:         time.Minute,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if got.Partial || len(got.Documents) != 2 {
		t.Fatalf("got %+v, want full 2 docs", got)
	}
	if got.Documents[0].ID != "service:2" {
		t.Fatalf("sort[0] = %q, want service:2 (newer issued_at)", got.Documents[0].ID)
	}
	if stored == nil || stored.Partial {
		t.Fatal("NFR6: full result must be cached")
	}
	if storeTTL != time.Minute {
		t.Fatalf("ttl = %s, want 1m", storeTTL)
	}
	if sourceReport(t, got.Sources, domain.SourceSales).Status != domain.SourceStatusOK {
		t.Fatalf("sales report = %+v, want OK", got.Sources)
	}
	if sourceReport(t, got.Sources, domain.SourceService).Status != domain.SourceStatusOK {
		t.Fatalf("service report = %+v, want OK", got.Sources)
	}
}

func TestFailureDoesNotCancelSibling(t *testing.T) {
	ctrl := gomock.NewController(t)
	var cancelled atomic.Bool
	healthy := delayedSource(ctrl, domain.SourceSales, 150*time.Millisecond, []domain.Document{salesDoc("1")}, nil, &cancelled)
	failing := stubSource(ctrl, domain.SourceService, nil, errors.New("service down"))
	a := newAgg(ctrl, false, healthy, failing)
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatalf("err = %v, want nil on partial", err)
	}
	if !got.Partial || len(got.Documents) != 1 {
		t.Fatalf("got %+v, want partial with sales docs", got)
	}
	if cancelled.Load() {
		t.Fatal("NFR1: healthy source ctx was cancelled by sibling failure")
	}
	if got.Documents[0].Source != domain.SourceSales {
		t.Fatalf("kept source = %s, want SALES", got.Documents[0].Source)
	}
	if sourceReport(t, got.Sources, domain.SourceSales).Status != domain.SourceStatusOK {
		t.Fatalf("sales report = %+v, want OK", got.Sources)
	}
	svc := sourceReport(t, got.Sources, domain.SourceService)
	if svc.Status != domain.SourceStatusError || svc.ErrorCode != domain.CodeUpstreamError {
		t.Fatalf("service report = %+v, want ERROR / UPSTREAM_ERROR", svc)
	}
}

func TestParallelFanOutUnderOneSecond(t *testing.T) {
	ctrl := gomock.NewController(t)
	a := newAgg(ctrl, true,
		delayedSource(ctrl, domain.SourceSales, 500*time.Millisecond, []domain.Document{salesDoc("1")}, nil, nil),
		delayedSource(ctrl, domain.SourceService, 500*time.Millisecond, []domain.Document{serviceDoc("2")}, nil, nil),
	)
	start := time.Now()
	got, err := a.Documents(context.Background(), testVIN)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if got.Partial || len(got.Documents) != 2 {
		t.Fatalf("got %+v, want both sources", got)
	}
	if elapsed >= time.Second {
		t.Fatalf("elapsed %s, want < 1s (NFR3 parallel)", elapsed)
	}
	if elapsed < 400*time.Millisecond {
		t.Fatalf("elapsed %s, sources did not wait ~500ms", elapsed)
	}
}

func TestBothFailWithoutCache(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	a := newAgg(ctrl, false,
		stubSource(ctrl, domain.SourceSales, nil, errors.New("down")),
		stubSource(ctrl, domain.SourceService, nil, context.DeadlineExceeded),
	)
	got, err := a.Documents(context.Background(), testVIN)
	if !errors.Is(err, domain.ErrAllSourcesUnavailable) {
		t.Fatalf("err = %v, want ErrAllSourcesUnavailable", err)
	}
	if sourceReport(t, got.Sources, domain.SourceSales).Status != domain.SourceStatusError {
		t.Fatalf("sales report = %+v, want ERROR", got.Sources)
	}
	if sourceReport(t, got.Sources, domain.SourceService).Status != domain.SourceStatusTimeout {
		t.Fatalf("service report = %+v, want TIMEOUT", got.Sources)
	}
}

func TestBothFailReturnsStaleCache(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	stale := domain.AggregateResult{Documents: []domain.Document{salesDoc("old")}, Partial: false}
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(domain.CachedResult{Result: stale, Stale: true}, true, nil)
	cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, nil, errors.New("down")),
			stubSource(ctrl, domain.SourceService, nil, errors.New("down")),
		},
		DocumentCache: cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Stale || got.ServedFromCache || len(got.Documents) != 1 {
		t.Fatalf("got %+v, want stale FR10 after fan-out, not a cache hit", got)
	}
	if sourceReport(t, got.Sources, domain.SourceSales).Status != domain.SourceStatusError {
		t.Fatalf("stale overlay sales = %+v, want live ERROR", got.Sources)
	}
	if sourceReport(t, got.Sources, domain.SourceService).Status != domain.SourceStatusError {
		t.Fatalf("stale overlay service = %+v, want live ERROR", got.Sources)
	}
}

func TestFreshCacheSkipsFanOut(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	cached := domain.AggregateResult{Documents: []domain.Document{salesDoc("c")}}
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(domain.CachedResult{Result: cached}, true, nil)
	cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	sales := appmocks.NewMockDocumentSource(ctrl)
	sales.EXPECT().Fetch(gomock.Any(), gomock.Any()).Times(0)
	sales.EXPECT().Name().Times(0)
	a := New(Options{
		Sources:       []app.DocumentSource{sales},
		DocumentCache: cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ServedFromCache || got.Documents[0].ID != "sales:c" {
		t.Fatalf("got %+v, want cached doc", got)
	}
}

func TestPartialCacheHitIsIgnored(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(
		domain.CachedResult{Result: domain.AggregateResult{
			Documents: []domain.Document{salesDoc("stale-partial")},
			Partial:   true,
		}}, true, nil)
	cache.EXPECT().Store(gomock.Any(), testVIN, gomock.Any(), gomock.Any()).Return(nil)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			stubSource(ctrl, domain.SourceService, []domain.Document{serviceDoc("2")}, nil),
		},
		DocumentCache: cache,
		CacheTTL:      time.Minute,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if got.ServedFromCache || got.Partial || len(got.Documents) != 2 {
		t.Fatalf("NFR6: got %+v, want live full fan-out", got)
	}
}

func TestCacheReadErrorFailsOpen(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(domain.CachedResult{}, false, errors.New("db down"))
	cache.EXPECT().Store(gomock.Any(), testVIN, gomock.Any(), gomock.Any()).Return(nil)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			stubSource(ctrl, domain.SourceService, []domain.Document{serviceDoc("2")}, nil),
		},
		DocumentCache: cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if got.ServedFromCache || len(got.Documents) != 2 {
		t.Fatalf("NFR7: got %+v, want live fan-out", got)
	}
}

func TestPartialIsNotCached(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	a := newAgg(ctrl, false,
		stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
		stubSource(ctrl, domain.SourceService, nil, errors.New("down")),
	)
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil || !got.Partial {
		t.Fatalf("got %+v err %v, want partial", got, err)
	}
}

func TestUnknownVINEmptyList(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	a := newAgg(ctrl, true,
		stubSource(ctrl, domain.SourceSales, nil, nil),
		stubSource(ctrl, domain.SourceService, nil, nil),
	)
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if got.Partial {
		t.Fatal("both empty sources is success, not partial")
	}
	if len(got.Documents) != 0 {
		t.Fatalf("docs = %d, want 0 (FR6)", len(got.Documents))
	}
}

func TestSourceTimeoutClassified(t *testing.T) {
	ctrl := gomock.NewController(t)
	cache := missCache(ctrl)
	cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			delayedSource(ctrl, domain.SourceService, 200*time.Millisecond, []domain.Document{serviceDoc("2")}, nil, nil),
		},
		PerSourceTimeout: 20 * time.Millisecond,
		DocumentCache:    cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Partial {
		t.Fatal("want partial when one source times out")
	}
	var timeout domain.SourceReport
	for _, s := range got.Sources {
		if s.Name == domain.SourceService {
			timeout = s
		}
	}
	if timeout.Status != domain.SourceStatusTimeout || timeout.ErrorCode != domain.CodeUpstreamTimeout {
		t.Fatalf("service report = %+v, want TIMEOUT / UPSTREAM_TIMEOUT", timeout)
	}
}

func TestSalesDownKeepsService(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	a := newAgg(ctrl, false,
		stubSource(ctrl, domain.SourceSales, nil, errors.New("down")),
		stubSource(ctrl, domain.SourceService, []domain.Document{serviceDoc("2")}, nil),
	)
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil || !got.Partial || len(got.Documents) != 1 {
		t.Fatalf("got %+v err %v, want partial service docs", got, err)
	}
	if got.Documents[0].Source != domain.SourceService {
		t.Fatalf("kept source = %s, want SERVICE", got.Documents[0].Source)
	}
	if sourceReport(t, got.Sources, domain.SourceSales).Status != domain.SourceStatusError {
		t.Fatalf("sales report = %+v, want ERROR", got.Sources)
	}
	if sourceReport(t, got.Sources, domain.SourceService).Status != domain.SourceStatusOK {
		t.Fatalf("service report = %+v, want OK", got.Sources)
	}
}

func TestTimeoutDoesNotCancelSibling(t *testing.T) {
	ctrl := gomock.NewController(t)
	var cancelled atomic.Bool
	cache := missCache(ctrl)
	cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	a := New(Options{
		Sources: []app.DocumentSource{
			delayedSource(ctrl, domain.SourceSales, 50*time.Millisecond, []domain.Document{salesDoc("1")}, nil, &cancelled),
			delayedSource(ctrl, domain.SourceService, 500*time.Millisecond, []domain.Document{serviceDoc("2")}, nil, nil),
		},
		PerSourceTimeout: 200 * time.Millisecond,
		DocumentCache:    cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Partial || len(got.Documents) != 1 {
		t.Fatalf("got %+v, want partial sales docs", got)
	}
	if cancelled.Load() {
		t.Fatal("NFR2: healthy source ctx was cancelled by sibling timeout")
	}
	if sourceReport(t, got.Sources, domain.SourceService).Status != domain.SourceStatusTimeout {
		t.Fatalf("service report = %+v, want TIMEOUT", got.Sources)
	}
}

func TestStaleCacheThenLiveFanOut(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	stale := domain.AggregateResult{Documents: []domain.Document{salesDoc("old")}, Partial: false}
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(domain.CachedResult{Result: stale, Stale: true}, true, nil)
	cache.EXPECT().Store(gomock.Any(), testVIN, gomock.Any(), gomock.Any()).Return(nil)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			stubSource(ctrl, domain.SourceService, []domain.Document{serviceDoc("2")}, nil),
		},
		DocumentCache: cache,
		CacheTTL:      time.Minute,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if got.Stale || got.ServedFromCache || got.Partial || len(got.Documents) != 2 {
		t.Fatalf("got %+v, want live full result recached", got)
	}
}

func TestStaleCacheOneLiveSource(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	stale := domain.AggregateResult{Documents: []domain.Document{salesDoc("old")}, Partial: false}
	cache := appmocks.NewMockDocumentCache(ctrl)
	cache.EXPECT().Lookup(gomock.Any(), testVIN).Return(domain.CachedResult{Result: stale, Stale: true}, true, nil)
	cache.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	a := New(Options{
		Sources: []app.DocumentSource{
			stubSource(ctrl, domain.SourceSales, []domain.Document{salesDoc("1")}, nil),
			stubSource(ctrl, domain.SourceService, nil, errors.New("down")),
		},
		DocumentCache: cache,
	})
	got, err := a.Documents(context.Background(), testVIN)
	if err != nil || !got.Partial || got.Stale || got.ServedFromCache || len(got.Documents) != 1 {
		t.Fatalf("got %+v err %v, want live partial not stale fallback", got, err)
	}
	if got.Documents[0].ID != "sales:1" {
		t.Fatalf("doc = %q, want live sales:1", got.Documents[0].ID)
	}
}

func TestCanceledRequestPropagates(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetchCanceled := func(ctx context.Context, _ string) ([]domain.Document, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	sales := appmocks.NewMockDocumentSource(ctrl)
	sales.EXPECT().Name().Return(domain.SourceSales).AnyTimes()
	sales.EXPECT().Fetch(gomock.Any(), testVIN).DoAndReturn(fetchCanceled)
	service := appmocks.NewMockDocumentSource(ctrl)
	service.EXPECT().Name().Return(domain.SourceService).AnyTimes()
	service.EXPECT().Fetch(gomock.Any(), testVIN).DoAndReturn(fetchCanceled)
	a := New(Options{
		Sources:       []app.DocumentSource{sales, service},
		DocumentCache: missCache(ctrl),
	})
	_, err := a.Documents(ctx, testVIN)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestNoSourcesReturnsUnavailable(t *testing.T) {
	t.Parallel()
	_, err := New(Options{}).Documents(context.Background(), testVIN)
	if !errors.Is(err, domain.ErrAllSourcesUnavailable) {
		t.Fatalf("err = %v, want ErrAllSourcesUnavailable", err)
	}
}
