// Package app holds audit use cases and the ports they depend on (R1).
//
//go:generate mockgen -source=$GOFILE -destination=mocks/mock_$GOFILE -package=${GOPACKAGE}mocks
package app

import (
	"context"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
)

// AccessRecorder is append-only. No update or delete method exists (NFR8).
type AccessRecorder interface {
	Record(ctx context.Context, e domain.AccessEvent) error
}
