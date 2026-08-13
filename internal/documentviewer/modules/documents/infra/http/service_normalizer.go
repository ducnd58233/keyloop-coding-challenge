package http

import (
	"strings"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

type serviceList struct {
	VehicleVIN  string              `json:"vehicleVin"`
	Attachments []serviceAttachment `json:"attachments"`
}

type serviceAttachment struct {
	AttachmentID string      `json:"attachmentId"`
	DocumentType string      `json:"documentType"`
	DisplayName  string      `json:"displayName"`
	IssuedDate   string      `json:"issuedDate"`
	File         serviceFile `json:"file"`
}

type serviceFile struct {
	URI string `json:"uri"`
}

func normalizeService(atts []serviceAttachment) []domain.Document {
	out := make([]domain.Document, 0, len(atts))
	for _, a := range atts {
		id := strings.TrimSpace(a.AttachmentID)
		if id == "" {
			continue
		}
		issued, err := time.Parse(time.RFC3339, strings.TrimSpace(a.IssuedDate))
		if err != nil {
			issued = time.Time{}
		} else {
			issued = issued.UTC()
		}
		out = append(out, domain.Document{
			ID:       namespaced(domain.SourceService, id),
			Source:   domain.SourceService,
			Type:     MapType(a.DocumentType),
			Title:    strings.TrimSpace(a.DisplayName),
			IssuedAt: issued,
			URL:      strings.TrimSpace(a.File.URI),
		})
	}
	return out
}
