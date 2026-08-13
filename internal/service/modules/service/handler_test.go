package service

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockfault"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
)

func TestListKnownVINShape(t *testing.T) {
	const base = "http://localhost:9101"
	rec := httptest.NewRecorder()
	New(Options{BaseURL: base}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/"+mockseed.WithDocumentsA+"/attachments", nil,
	))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["vehicleVin"] != mockseed.WithDocumentsA {
		t.Fatalf("vehicleVin = %v, want %s", body["vehicleVin"], mockseed.WithDocumentsA)
	}
	if _, exists := body["vin"]; exists {
		t.Fatal("service payload must not use sales key vin")
	}
	atts, ok := body["attachments"].([]any)
	if !ok || len(atts) == 0 {
		t.Fatalf("attachments = %v, want non-empty array", body["attachments"])
	}
	first, ok := atts[0].(map[string]any)
	if !ok {
		t.Fatalf("attachments[0] = %T, want object", atts[0])
	}
	for _, key := range []string{"attachmentId", "documentType", "displayName", "issuedDate", "file"} {
		if _, exists := first[key]; !exists {
			t.Fatalf("missing camelCase field %q in %v", key, first)
		}
	}
	docType, _ := first["documentType"].(string)
	if !isScreamingSnake(docType) {
		t.Fatalf("documentType = %q, want SCREAMING_SNAKE", docType)
	}
	issued, _ := first["issuedDate"].(string)
	if _, err := time.Parse(time.RFC3339, issued); err != nil {
		t.Fatalf("issuedDate = %q, want RFC3339: %v", issued, err)
	}
	file, ok := first["file"].(map[string]any)
	if !ok {
		t.Fatalf("file = %T, want nested object", first["file"])
	}
	uri, _ := file["uri"].(string)
	if !strings.HasPrefix(uri, base+"/service/v1/attachments/") {
		t.Fatalf("file.uri = %q, want absolute url under %s/service/v1/attachments/", uri, base)
	}
}

func TestListEmptyAndUnknownVIN(t *testing.T) {
	h := New(Options{})
	cases := []string{mockseed.WithNone, "UNKNOWNVIN"}
	for _, vin := range cases {
		t.Run(vin, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(
				http.MethodGet, "/service/v1/vehicles/"+vin+"/attachments", nil,
			))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var body struct {
				Attachments []json.RawMessage `json:"attachments"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Attachments) != 0 {
				t.Fatalf("attachments len = %d, want 0", len(body.Attachments))
			}
		})
	}
}

func TestSecondVINHasDocuments(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Options{}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/"+mockseed.WithDocumentsB+"/attachments", nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Attachments []json.RawMessage `json:"attachments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Attachments) == 0 {
		t.Fatal("WithDocumentsB must have at least one service attachment")
	}
}

func TestHealthzAndOutage(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Options{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	New(Options{Fault: mockfault.Config{Down: true}}).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/healthz", nil),
	)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("down healthz status = %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	New(Options{Fault: mockfault.Config{Down: true}}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/"+mockseed.WithDocumentsA+"/attachments", nil,
	))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("down list status = %d, want 503", rec.Code)
	}
}

func TestListLogsVINSuffixNotFullVIN(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	rec := httptest.NewRecorder()
	New(Options{Log: log}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/"+mockseed.WithDocumentsA+"/attachments", nil,
	))
	joined := buf.String()
	if strings.Contains(joined, mockseed.WithDocumentsA) {
		t.Fatalf("log leaked full VIN: %s", joined)
	}
	if !strings.Contains(joined, mockseed.Suffix(mockseed.WithDocumentsA)) {
		t.Fatalf("log missing vin_suffix: %s", joined)
	}
	if strings.Contains(joined, "/vehicles/"+mockseed.WithDocumentsA) {
		t.Fatalf("log leaked VIN in route: %s", joined)
	}
}

func TestCatalogCoversSeed(t *testing.T) {
	docs := catalog(defaultBaseURL)
	if len(docs) < 20 {
		t.Fatalf("catalog size = %d, want >= 20", len(docs))
	}
	for _, vin := range mockseed.All {
		atts, ok := docs[vin.Value]
		if !ok {
			t.Fatalf("missing catalog entry for %s", vin.Value)
		}
		has := len(atts) > 0
		if mockseed.HasService(vin.Kind) != has {
			t.Fatalf("%s kind=%s attachments=%d", vin.Value, vin.Kind, len(atts))
		}
	}
}

func TestGenerateUnknownVIN(t *testing.T) {
	seq := 0
	intN := func(n int) int {
		seq++
		if n <= 0 {
			return 0
		}
		return seq % n
	}
	rec := httptest.NewRecorder()
	New(Options{Generate: true, IntN: intN}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/UNKNOWNVIN/attachments", nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Attachments []json.RawMessage `json:"attachments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Attachments) == 0 {
		t.Fatal("Generate=true must invent attachments for unknown VINs")
	}

	rec = httptest.NewRecorder()
	New(Options{Generate: true, IntN: intN}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/service/v1/vehicles/"+mockseed.WithNone+"/attachments", nil,
	))
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Attachments) != 0 {
		t.Fatal("WithNone must stay empty even when Generate=true")
	}
}

func isScreamingSnake(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == '_' {
			continue
		}
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
