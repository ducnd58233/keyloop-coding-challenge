package http

import (
	"strings"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

type salesList struct {
	VIN     string        `json:"vin"`
	Records []salesRecord `json:"records"`
}

type salesRecord struct {
	DocID        string `json:"doc_id"`
	Category     string `json:"category"`
	Name         string `json:"name"`
	CreatedEpoch int64  `json:"created_epoch"`
	DownloadPath string `json:"download_path"`
}

func normalizeSales(base string, recs []salesRecord) []domain.Document {
	out := make([]domain.Document, 0, len(recs))
	for _, rec := range recs {
		id := strings.TrimSpace(rec.DocID)
		if id == "" {
			continue
		}
		var issued time.Time
		if rec.CreatedEpoch != 0 {
			issued = time.Unix(rec.CreatedEpoch, 0).UTC()
		}
		out = append(out, domain.Document{
			ID:       namespaced(domain.SourceSales, id),
			Source:   domain.SourceSales,
			Type:     MapType(rec.Category),
			Title:    strings.TrimSpace(rec.Name),
			IssuedAt: issued,
			URL:      resolveURL(base, rec.DownloadPath),
		})
	}
	return out
}
