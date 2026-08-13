package http

import (
	"context"
	"encoding/json"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

func TestServiceClientNormalisesRFC3339AndNestedURI(t *testing.T) {
	t.Parallel()
	const abs = "http://localhost:9101/service/v1/attachments/WO-77/raw"
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if !strings.HasSuffix(r.URL.Path, "/service/v1/vehicles/"+testVIN+"/attachments") &&
			r.URL.Path != "/service/v1/vehicles/"+testVIN+"/attachments" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serviceList{
			VehicleVIN: testVIN,
			Attachments: []serviceAttachment{{
				AttachmentID: "WO-77",
				DocumentType: "WORK_ORDER",
				DisplayName:  "60,000 mile service",
				IssuedDate:   "2025-01-04T09:30:00Z",
				File:         serviceFile{URI: abs},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	docs, err := NewServiceClient(srv.URL).Fetch(context.Background(), testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("len = %d", len(docs))
	}
	d := docs[0]
	if d.ID != "service:WO-77" || d.Source != domain.SourceService || d.Type != "WORK_ORDER" {
		t.Fatalf("doc = %+v", d)
	}
	if d.Title != "60,000 mile service" || d.URL != abs {
		t.Fatalf("title/url = %q %q", d.Title, d.URL)
	}
	want := time.Date(2025, 1, 4, 9, 30, 0, 0, time.UTC)
	if !d.IssuedAt.Equal(want) {
		t.Fatalf("issued_at = %s", d.IssuedAt)
	}
}

func TestServiceClientInvalidDateStillReturnsDoc(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vehicleVin":"1HGCM82633","attachments":[{"attachmentId":"WO-1","documentType":"WORK_ORDER","displayName":"x","issuedDate":"not-a-date","file":{"uri":"http://example/a"}}]}`)
	}))
	t.Cleanup(srv.Close)
	docs, err := NewServiceClient(srv.URL).Fetch(context.Background(), testVIN)
	if err != nil || len(docs) != 1 || !docs[0].IssuedAt.IsZero() {
		t.Fatalf("docs=%v err=%v", docs, err)
	}
}

func TestServiceClient5xxIsErrorWithoutVIN(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	_, err := NewServiceClient(srv.URL).Fetch(context.Background(), testVIN)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), testVIN) {
		t.Fatalf("error leaked vin: %v", err)
	}
}
