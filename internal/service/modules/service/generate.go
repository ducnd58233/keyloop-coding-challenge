package service

import (
	"fmt"
	"time"

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
	maxExtraAttachments = 4
	issuedWindowHours   = 400 * 24
)

func generateAttachments(vin, baseURL string, intN func(int) int) ([]attachment, error) {
	count, err := pickN(intN, maxExtraAttachments)
	if err != nil {
		return nil, err
	}
	n := 1 + count
	out := make([]attachment, n)
	suffix := mockseed.Suffix(vin)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		typeIdx, err := pickN(intN, len(serviceTypes))
		if err != nil {
			return nil, err
		}
		hours, err := pickN(intN, issuedWindowHours)
		if err != nil {
			return nil, err
		}
		docType := serviceTypes[typeIdx]
		id := fmt.Sprintf("%s-%s-%d", docType[:3], suffix, i+1)
		issued := base.Add(time.Duration(hours) * time.Hour)
		out[i] = attachment{
			AttachmentID: id,
			DocumentType: docType,
			DisplayName:  fmt.Sprintf("%s record %d", docType, i+1),
			IssuedDate:   issued.Format(time.RFC3339),
			File:         file{URI: baseURL + "/service/v1/attachments/" + id + "/raw"},
		}
	}
	return out, nil
}
