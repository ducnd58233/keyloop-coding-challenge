package sales

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockfault"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

func TestListKnownVINShape(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Options{}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithDocumentsA, nil,
	))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["vin"] != mockseed.WithDocumentsA {
		t.Fatalf("vin = %v, want %s", body["vin"], mockseed.WithDocumentsA)
	}
	records, ok := body["records"].([]any)
	if !ok || len(records) == 0 {
		t.Fatalf("records = %v, want non-empty array", body["records"])
	}
	first, ok := records[0].(map[string]any)
	if !ok {
		t.Fatalf("record[0] = %T, want object", records[0])
	}
	for _, key := range []string{"doc_id", "category", "name", "created_epoch", "download_path"} {
		if _, exists := first[key]; !exists {
			t.Fatalf("missing snake_case field %q in %v", key, first)
		}
	}
	if _, exists := first["docId"]; exists {
		t.Fatal("sales payload must not use camelCase docId")
	}
	category, _ := first["category"].(string)
	if category == "" || !allLower(category) {
		t.Fatalf("category = %q, want lowercase type vocabulary", category)
	}
	if _, ok := first["created_epoch"].(float64); !ok {
		t.Fatalf("created_epoch = %T, want number (epoch seconds)", first["created_epoch"])
	}
	path, _ := first["download_path"].(string)
	if !strings.HasPrefix(path, "/sales/v1/documents/") {
		t.Fatalf("download_path = %q, want relative /sales/v1/documents/...", path)
	}
}

func TestListEmptyAndUnknownVIN(t *testing.T) {
	h := New(Options{})
	cases := []string{mockseed.WithNone, "UNKNOWNVIN"}
	for _, vin := range cases {
		t.Run(vin, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(
				http.MethodGet, "/sales/v1/documents?vin="+vin, nil,
			))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var body struct {
				Records []json.RawMessage `json:"records"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Records) != 0 {
				t.Fatalf("records len = %d, want 0", len(body.Records))
			}
		})
	}
}

func TestSecondVINHasDocuments(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Options{}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithDocumentsB, nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Records) == 0 {
		t.Fatal("WithDocumentsB must have at least one sales record")
	}
}

func TestHealthzAndOutage(t *testing.T) {
	rec := httptest.NewRecorder()
	New(Options{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	New(Options{Fault: mockfault.Config{Down: true}}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("down healthz status = %d, want 503", rec.Code)
	}

	rec = httptest.NewRecorder()
	New(Options{Fault: mockfault.Config{Down: true}}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithDocumentsA, nil,
	))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("down list status = %d, want 503", rec.Code)
	}
}

func TestCatalogCoversSeed(t *testing.T) {
	if len(catalog) < 20 {
		t.Fatalf("catalog size = %d, want >= 20", len(catalog))
	}
	for _, vin := range mockseed.All {
		recs, ok := catalog[vin.Value]
		if !ok {
			t.Fatalf("missing catalog entry for %s", vin.Value)
		}
		has := len(recs) > 0
		if mockseed.HasSales(vin.Kind) != has {
			t.Fatalf("%s kind=%s records=%d", vin.Value, vin.Kind, len(recs))
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
		http.MethodGet, "/sales/v1/documents?vin=UNKNOWNVIN", nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Records) == 0 {
		t.Fatal("Generate=true must invent records for unknown VINs")
	}

	rec = httptest.NewRecorder()
	New(Options{Generate: true, IntN: intN}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithNone, nil,
	))
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Records) != 0 {
		t.Fatal("WithNone must stay empty even when Generate=true")
	}
}

func TestListLogsVINSuffixNotFullVIN(t *testing.T) {
	log := &observability.Capture{}
	rec := httptest.NewRecorder()
	New(Options{Log: log}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithDocumentsA, nil,
	))
	joined := log.Text()
	if strings.Contains(joined, mockseed.WithDocumentsA) {
		t.Fatalf("log leaked full VIN: %s", joined)
	}
	if !strings.Contains(joined, mockseed.Suffix(mockseed.WithDocumentsA)) {
		t.Fatalf("log missing vin_suffix: %s", joined)
	}
	if !strings.Contains(joined, "fault=ok") {
		t.Fatalf("log missing fault=ok: %s", joined)
	}
}

func TestListLogsRandomLatency(t *testing.T) {
	log := &observability.Capture{}
	rec := httptest.NewRecorder()
	New(Options{
		Log: log,
		Fault: mockfault.Config{
			LatencyRate:    1,
			Random:         func() float64 { return 0 },
			Sleep:          func(time.Duration) {},
			RandomDuration: func() time.Duration { return 400 * time.Millisecond },
		},
	}).ServeHTTP(rec, httptest.NewRequest(
		http.MethodGet, "/sales/v1/documents?vin="+mockseed.WithDocumentsA, nil,
	))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 after latency", rec.Code)
	}
	joined := log.Text()
	if !strings.Contains(joined, "fault=latency") || !strings.Contains(joined, "latency=400ms") {
		t.Fatalf("log missing latency fields: %s", joined)
	}
}

func allLower(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return false
		}
	}
	return true
}
