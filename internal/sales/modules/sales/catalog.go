// Package sales is the Sales System mock: flat snake_case payloads on :9100 (A5).
package sales

import (
	"fmt"
	"strings"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
)

type record struct {
	DocID        string `json:"doc_id"`
	Category     string `json:"category"`
	Name         string `json:"name"`
	CreatedEpoch int64  `json:"created_epoch"`
	DownloadPath string `json:"download_path"`
}

type listResponse struct {
	VIN     string   `json:"vin"`
	Records []record `json:"records"`
}

var salesCategories = []string{"invoice", "finance", "registration", "contract", "quote"}

var catalog = buildCatalog()

func buildCatalog() map[string][]record {
	out := make(map[string][]record, len(mockseed.All))
	for i, vin := range mockseed.All {
		switch vin.Value {
		case mockseed.WithDocumentsA:
			out[vin.Value] = []record{
				{
					DocID:        "INV-1001",
					Category:     "invoice",
					Name:         "Purchase Invoice",
					CreatedEpoch: 1710115200,
					DownloadPath: "/sales/v1/documents/INV-1001/raw",
				},
				{
					DocID:        "QTE-1002",
					Category:     "quote",
					Name:         "Part-ex Quote",
					CreatedEpoch: 1709251200,
					DownloadPath: "/sales/v1/documents/QTE-1002/raw",
				},
			}
		case mockseed.WithDocumentsB:
			out[vin.Value] = []record{
				{
					DocID:        "FIN-2201",
					Category:     "finance",
					Name:         "Hire Purchase Agreement",
					CreatedEpoch: 1704067200,
					DownloadPath: "/sales/v1/documents/FIN-2201/raw",
				},
				{
					DocID:        "REG-9",
					Category:     "registration",
					Name:         "V5C Registration",
					CreatedEpoch: 1693526400,
					DownloadPath: "/sales/v1/documents/REG-9/raw",
				},
				{
					DocID:        "CON-44",
					Category:     "contract",
					Name:         "Sale Contract",
					CreatedEpoch: 1696118400,
					DownloadPath: "/sales/v1/documents/CON-44/raw",
				},
			}
		default:
			if mockseed.HasSales(vin.Kind) {
				out[vin.Value] = seedRecords(vin.Value, i)
			} else {
				out[vin.Value] = []record{}
			}
		}
	}
	return out
}

func seedRecords(vin string, idx int) []record {
	n := 1 + idx%3
	suffix := mockseed.Suffix(vin)
	base := int64(1_700_000_000)
	out := make([]record, n)
	for j := 0; j < n; j++ {
		cat := salesCategories[(idx+j)%len(salesCategories)]
		id := fmt.Sprintf("%s-%s-%d", strings.ToUpper(cat[:3]), suffix, j+1)
		out[j] = record{
			DocID:        id,
			Category:     cat,
			Name:         fmt.Sprintf("%s document %d", cat, j+1),
			CreatedEpoch: base + int64((idx*1000+j)*86400),
			DownloadPath: "/sales/v1/documents/" + id + "/raw",
		}
	}
	return out
}
