package persistence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func TestStoreRejectsPartialWithoutIO(t *testing.T) {
	t.Parallel()
	repo := NewCacheRepository(nil, common.SystemClock{})
	err := repo.Store(context.Background(), testutil.TestVIN, domain.AggregateResult{
		Partial: true,
		Documents: []domain.Document{{
			ID:     "sales:1",
			Source: domain.SourceSales,
		}},
	}, time.Minute)
	if !errors.Is(err, ErrPartialNotCached) {
		t.Fatalf("Store(partial) err = %v, want ErrPartialNotCached", err)
	}
}
