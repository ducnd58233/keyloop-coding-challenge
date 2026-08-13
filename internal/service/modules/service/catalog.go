// Package service is the Service System mock: nested camelCase payloads on :9101 (A5).
package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
)

const defaultBaseURL = "http://localhost:9101"

type file struct {
	URI string `json:"uri"`
}

type attachment struct {
	AttachmentID string `json:"attachmentId"`
	DocumentType string `json:"documentType"`
	DisplayName  string `json:"displayName"`
	IssuedDate   string `json:"issuedDate"`
	File         file   `json:"file"`
}

type listResponse struct {
	VehicleVIN  string       `json:"vehicleVin"`
	Attachments []attachment `json:"attachments"`
}

var serviceTypes = []string{"WORK_ORDER", "INSPECTION", "WARRANTY", "RECALL", "MOT"}

func catalog(baseURL string) map[string][]attachment {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	out := make(map[string][]attachment, len(mockseed.All))
	for i, vin := range mockseed.All {
		switch vin.Value {
		case mockseed.WithDocumentsA:
			out[vin.Value] = []attachment{
				{
					AttachmentID: "WO-77",
					DocumentType: "WORK_ORDER",
					DisplayName:  "60,000 mile service",
					IssuedDate:   "2025-01-04T09:30:00Z",
					File:         file{URI: base + "/service/v1/attachments/WO-77/raw"},
				},
				{
					AttachmentID: "WAR-3",
					DocumentType: "WARRANTY",
					DisplayName:  "Powertrain warranty claim",
					IssuedDate:   "2024-08-12T11:00:00Z",
					File:         file{URI: base + "/service/v1/attachments/WAR-3/raw"},
				},
			}
		case mockseed.WithDocumentsB:
			out[vin.Value] = []attachment{
				{
					AttachmentID: "INSP-12",
					DocumentType: "INSPECTION",
					DisplayName:  "Annual inspection",
					IssuedDate:   "2024-11-02T14:00:00Z",
					File:         file{URI: base + "/service/v1/attachments/INSP-12/raw"},
				},
				{
					AttachmentID: "MOT-8",
					DocumentType: "MOT",
					DisplayName:  "MOT certificate",
					IssuedDate:   "2024-06-01T08:15:00Z",
					File:         file{URI: base + "/service/v1/attachments/MOT-8/raw"},
				},
			}
		default:
			if mockseed.HasService(vin.Kind) {
				out[vin.Value] = seedAttachments(vin.Value, base, i)
			} else {
				out[vin.Value] = []attachment{}
			}
		}
	}
	return out
}

func seedAttachments(vin, base string, idx int) []attachment {
	n := 1 + idx%3
	suffix := mockseed.Suffix(vin)
	day := time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)
	out := make([]attachment, n)
	for j := 0; j < n; j++ {
		docType := serviceTypes[(idx+j)%len(serviceTypes)]
		id := fmt.Sprintf("%s-%s-%d", docType[:3], suffix, j+1)
		issued := day.AddDate(0, 0, idx+j*7)
		out[j] = attachment{
			AttachmentID: id,
			DocumentType: docType,
			DisplayName:  fmt.Sprintf("%s record %d", docType, j+1),
			IssuedDate:   issued.Format(time.RFC3339),
			File:         file{URI: base + "/service/v1/attachments/" + id + "/raw"},
		}
	}
	return out
}
