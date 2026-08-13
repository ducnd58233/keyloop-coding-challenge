package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
)

const testVIN = "1HGCM82633"

func TestSalesClientNormalisesEpochAndRelativeURL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path != "/sales/v1/documents" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("vin") != testVIN {
			t.Fatalf("vin query leaked or missing")
		}
		if r.Header.Get("X-Request-Id") != "req-sales" {
			t.Fatalf("X-Request-Id = %q", r.Header.Get("X-Request-Id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(salesList{
			VIN: testVIN,
			Records: []salesRecord{{
				DocID:        "INV-1001",
				Category:     "invoice",
				Name:         "Purchase Invoice",
				CreatedEpoch: 1710115200,
				DownloadPath: "/sales/v1/documents/INV-1001/raw",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewSalesClient(srv.URL)
	ctx := httpserver.WithRequestID(context.Background(), "req-sales")
	docs, err := c.Fetch(ctx, testVIN)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("len = %d", len(docs))
	}
	d := docs[0]
	if d.ID != "sales:INV-1001" || d.Source != domain.SourceSales || d.Type != "INVOICE" || d.Title != "Purchase Invoice" {
		t.Fatalf("doc = %+v", d)
	}
	if !d.IssuedAt.Equal(time.Unix(1710115200, 0).UTC()) {
		t.Fatalf("issued_at = %s", d.IssuedAt)
	}
	wantURL := srv.URL + "/sales/v1/documents/INV-1001/raw"
	if d.URL != wantURL {
		t.Fatalf("url = %q, want %q", d.URL, wantURL)
	}
}

func TestSalesClientUnknownTypeIsOTHER(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vin":"1HGCM82633","records":[{"doc_id":"X-1","category":"mystery","name":"X","created_epoch":1,"download_path":"/x"}]}`)
	}))
	t.Cleanup(srv.Close)
	docs, err := NewSalesClient(srv.URL).Fetch(context.Background(), testVIN)
	if err != nil || len(docs) != 1 || docs[0].Type != "OTHER" {
		t.Fatalf("docs=%v err=%v", docs, err)
	}
}

func TestSalesClient5xxIsErrorWithoutVIN(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.WriteHeader(nethttp.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":"`+testVIN+`"}`)
	}))
	t.Cleanup(srv.Close)
	_, err := NewSalesClient(srv.URL).Fetch(context.Background(), testVIN)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), testVIN) || strings.Contains(err.Error(), "localhost") {
		t.Fatalf("error leaked vin or host: %v", err)
	}
}

func TestSalesClientHonoursContextTimeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, r *nethttp.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := NewSalesClient(srv.URL).Fetch(ctx, testVIN)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if strings.Contains(err.Error(), testVIN) || strings.Contains(err.Error(), "localhost") || strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("timeout error leaked vin or host: %v", err)
	}
}

func TestSalesClientEmptyRecords(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vin":"3N1AB7AP1D","records":[]}`)
	}))
	t.Cleanup(srv.Close)
	docs, err := NewSalesClient(srv.URL).Fetch(context.Background(), "3N1AB7AP1D")
	if err != nil || docs == nil || len(docs) != 0 {
		t.Fatalf("FR6 empty: docs=%v err=%v", docs, err)
	}
}
