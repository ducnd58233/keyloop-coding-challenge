package sales

import (
	"fmt"
	"strings"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/randutil"
)

func pickN(intN func(int) int, n int) (int, error) {
	if intN != nil {
		return intN(n), nil
	}
	return randutil.IntN(n)
}

func generateRecords(vin string, intN func(int) int) ([]record, error) {
	count, err := pickN(intN, 4)
	if err != nil {
		return nil, err
	}
	n := 1 + count
	out := make([]record, n)
	suffix := mockseed.Suffix(vin)
	baseEpoch := int64(1_700_000_000)
	for i := 0; i < n; i++ {
		catIdx, err := pickN(intN, len(salesCategories))
		if err != nil {
			return nil, err
		}
		cat := salesCategories[catIdx]
		offset, err := pickN(intN, 20_000_000)
		if err != nil {
			return nil, err
		}
		id := fmt.Sprintf("%s-%s-%d", strings.ToUpper(cat[:3]), suffix, i+1)
		out[i] = record{
			DocID:        id,
			Category:     cat,
			Name:         fmt.Sprintf("%s document %d", cat, i+1),
			CreatedEpoch: baseEpoch + int64(offset),
			DownloadPath: "/sales/v1/documents/" + id + "/raw",
		}
	}
	return out, nil
}
