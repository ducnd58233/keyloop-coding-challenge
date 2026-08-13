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

const (
	maxExtraRecords = 4
	seedEpoch       = int64(1_700_000_000)
	epochJitterMax  = 20_000_000
	secondsPerDay   = 86400
)

func generateRecords(vin string, intN func(int) int) ([]record, error) {
	count, err := pickN(intN, maxExtraRecords)
	if err != nil {
		return nil, err
	}
	n := 1 + count
	out := make([]record, n)
	suffix := mockseed.Suffix(vin)
	baseEpoch := seedEpoch
	for i := 0; i < n; i++ {
		catIdx, err := pickN(intN, len(salesCategories))
		if err != nil {
			return nil, err
		}
		cat := salesCategories[catIdx]
		offset, err := pickN(intN, epochJitterMax)
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
