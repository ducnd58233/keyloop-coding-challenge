package usecases

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	auditapp "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app"
	appmocks "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app/mocks"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/testutil"
)

func TestAccessRecorderIsAppendOnly(t *testing.T) {
	t.Parallel()
	var rec auditapp.AccessRecorder
	iface := reflect.TypeOf(&rec).Elem()
	if iface.NumMethod() != 1 || iface.Method(0).Name != "Record" {
		t.Fatalf("AccessRecorder must expose only Record (NFR8), got %d methods", iface.NumMethod())
	}
}

func TestRecordAccessHashesVINAndUsesClock(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	rec := appmocks.NewMockAccessRecorder(ctrl)
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	clock := common.NewFixedClock(now)
	const vin = testutil.TestVIN
	const salt = "unit-test-salt"
	wantHash, wantSuffix := common.HashVIN(salt, vin)

	var got domain.AccessEvent
	rec.EXPECT().Record(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, e domain.AccessEvent) error {
			got = e
			return nil
		},
	)

	u := NewRecordAccess(rec, salt, clock)
	err := u.Execute(context.Background(), AccessInput{
		VIN:           vin,
		ActorID:       "tech-1",
		RequestID:     "req-1",
		TraceID:       "tr-1",
		Outcome:       domain.OutcomeInvalidVIN,
		SourcesOK:     0,
		SourcesFailed: 0,
		Latency:       12 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.VINHash != wantHash || got.VINSuffix != wantSuffix {
		t.Fatalf("hash/suffix = %q %q, want %q %q", got.VINHash, got.VINSuffix, wantHash, wantSuffix)
	}
	if got.VINHash == vin || got.VINSuffix == vin {
		t.Fatal("full VIN must not reach the audit store")
	}
	if got.Outcome != domain.OutcomeInvalidVIN || got.ActorID != "tech-1" {
		t.Fatalf("event = %+v", got)
	}
	if !got.RequestedAt.Equal(now) {
		t.Fatalf("RequestedAt = %s, want fake clock %s", got.RequestedAt, now)
	}
	if got.Latency != 12*time.Millisecond {
		t.Fatalf("Latency = %s", got.Latency)
	}
}
