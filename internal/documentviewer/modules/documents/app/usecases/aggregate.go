// Package usecases holds documents application flows with no I/O of their own.
package usecases

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

// Aggregate looks up documents for one VIN. Zero I/O: sources and cache are ports.
type Aggregate struct {
	sources          []app.DocumentSource
	docsCache        app.DocumentCache
	vin              domain.Format
	perSourceTimeout time.Duration
	aggregateTimeout time.Duration
	cacheTTL         time.Duration
}

// Options wires timeouts from configs, not from the environment.
type Options struct {
	Sources          []app.DocumentSource
	DocumentCache    app.DocumentCache
	VIN              domain.Format
	PerSourceTimeout time.Duration
	AggregateTimeout time.Duration
	CacheTTL         time.Duration
}

// New applies A1 DefaultFormat when Options.VIN is zero.
func New(opt Options) *Aggregate {
	vin := opt.VIN
	if vin.Length == 0 || vin.Alphabet == "" {
		vin = domain.DefaultFormat
	}
	return &Aggregate{
		sources:          opt.Sources,
		docsCache:        opt.DocumentCache,
		vin:              vin,
		perSourceTimeout: opt.PerSourceTimeout,
		aggregateTimeout: opt.AggregateTimeout,
		cacheTTL:         opt.CacheTTL,
	}
}

type fetchOutcome struct {
	name domain.SourceName
	docs []domain.Document
	err  error
}

// Documents fans out to every source. A source failure is recorded as data so
// it cannot cancel a healthy sibling (NFR1 / DD-2).
func (a *Aggregate) Documents(ctx context.Context, vin string) (domain.AggregateResult, error) {
	if !a.vin.Valid(vin) {
		return domain.AggregateResult{}, domain.ErrInvalidVIN
	}

	cached, staleFallback := a.lookupCache(ctx, vin)
	if cached != nil {
		return *cached, nil
	}
	if len(a.sources) == 0 {
		return domain.AggregateResult{}, domain.ErrAllSourcesUnavailable
	}

	aggCtx := ctx
	cancelAgg := func() {}
	if a.aggregateTimeout > 0 {
		aggCtx, cancelAgg = context.WithTimeout(ctx, a.aggregateTimeout)
	}
	defer cancelAgg()

	outcomes := a.fanOut(aggCtx, vin)
	if err := ctx.Err(); err != nil {
		return domain.AggregateResult{}, err
	}
	reports, docs, okCount := classify(outcomes)

	switch okCount {
	case len(a.sources):
		result := domain.AggregateResult{
			Documents: domain.Merge(docs),
			Sources:   reports,
			Partial:   false,
		}
		a.storeFull(ctx, vin, result)
		return result, nil
	case 0:
		if staleFallback != nil {
			out := *staleFallback
			out.Stale = true
			out.ServedFromCache = false
			out.Sources = reports
			return out, nil
		}
		return domain.AggregateResult{Sources: reports}, domain.ErrAllSourcesUnavailable
	default:
		return domain.AggregateResult{
			Documents: domain.Merge(docs),
			Sources:   reports,
			Partial:   true,
		}, nil
	}
}

func (a *Aggregate) lookupCache(ctx context.Context, vin string) (fresh *domain.AggregateResult, stale *domain.AggregateResult) {
	if a.docsCache == nil {
		return nil, nil
	}
	hit, ok, err := a.docsCache.Lookup(ctx, vin)
	if err != nil || !ok {
		return nil, nil
	}
	res := hit.Result
	if res.Partial {
		return nil, nil
	}
	if hit.Stale || hit.Result.Stale {
		res.Stale = true
		return nil, &res
	}
	res.ServedFromCache = true
	return &res, nil
}

func (a *Aggregate) storeFull(ctx context.Context, vin string, result domain.AggregateResult) {
	if a.docsCache == nil || result.Partial {
		return
	}
	_ = a.docsCache.Store(ctx, vin, result, a.cacheTTL)
}

func (a *Aggregate) fanOut(ctx context.Context, vin string) []fetchOutcome {
	out := make([]fetchOutcome, len(a.sources))
	g, gctx := errgroup.WithContext(ctx)
	for i, src := range a.sources {
		g.Go(func() error {
			srcCtx := gctx
			cancel := func() {}
			if a.perSourceTimeout > 0 {
				srcCtx, cancel = context.WithTimeout(gctx, a.perSourceTimeout)
			}
			defer cancel()
			docs, err := src.Fetch(srcCtx, vin)
			out[i] = fetchOutcome{name: src.Name(), docs: docs, err: err}
			return nil
		})
	}
	_ = g.Wait()
	return out
}

func classify(outcomes []fetchOutcome) (reports []domain.SourceReport, docs []domain.Document, okCount int) {
	reports = make([]domain.SourceReport, 0, len(outcomes))
	for _, o := range outcomes {
		rep := domain.SourceReport{Name: o.name, Status: domain.SourceStatusOK}
		if o.err != nil {
			rep.Status = domain.SourceStatusError
			rep.ErrorCode = domain.CodeUpstreamError
			if errors.Is(o.err, context.DeadlineExceeded) {
				rep.Status = domain.SourceStatusTimeout
				rep.ErrorCode = domain.CodeUpstreamTimeout
			}
			reports = append(reports, rep)
			continue
		}
		okCount++
		docs = append(docs, o.docs...)
		reports = append(reports, rep)
	}
	return reports, docs, okCount
}
