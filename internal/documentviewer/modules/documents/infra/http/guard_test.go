package http

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/circuitbreaker"
)

type countingSource struct {
	name domain.SourceName
	n    atomic.Int32
	err  error
	docs []domain.Document
}

func (s *countingSource) Name() domain.SourceName { return s.name }

func (s *countingSource) Fetch(context.Context, string) ([]domain.Document, error) {
	s.n.Add(1)
	return s.docs, s.err
}

func TestWithBreakerFailFastAfterOpen(t *testing.T) {
	t.Parallel()
	inner := &countingSource{name: domain.SourceSales, err: errors.New("upstream fetch failed")}
	cb := circuitbreaker.New(circuitbreaker.Settings{Name: "sales", Threshold: 2, Cooldown: time.Minute})
	src := WithBreaker(inner, cb)
	_, _ = src.Fetch(context.Background(), "1HGCM82633")
	_, _ = src.Fetch(context.Background(), "1HGCM82633")
	_, err := src.Fetch(context.Background(), "1HGCM82633")
	if !errors.Is(err, circuitbreaker.ErrOpen) {
		t.Fatalf("err = %v, want ErrOpen", err)
	}
	if inner.n.Load() != 2 {
		t.Fatalf("inner calls = %d, want 2", inner.n.Load())
	}
	if src.Name() != domain.SourceSales {
		t.Fatalf("name = %s", src.Name())
	}
}

func TestWithBreakerNilIsPassthrough(t *testing.T) {
	t.Parallel()
	inner := &countingSource{name: domain.SourceService, docs: []domain.Document{{ID: "1"}}}
	src := WithBreaker(inner, nil)
	docs, err := src.Fetch(context.Background(), "1HGCM82633")
	if err != nil || len(docs) != 1 || inner.n.Load() != 1 {
		t.Fatalf("passthrough: docs=%v err=%v n=%d", docs, err, inner.n.Load())
	}
}
