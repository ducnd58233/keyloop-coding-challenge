package openapidocs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubSpec struct {
	body string
}

func (s stubSpec) ReadDoc() string { return s.body }

func TestMountServesJSONAndUI(t *testing.T) {
	mux := http.NewServeMux()
	Mount(mux, stubSpec{`{"openapi":"3.1.0","info":{"title":"t","version":"1"}}`})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := srv.Client()

	res, err := client.Get(srv.URL + PathJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("json status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != contentJSON {
		t.Fatalf("json content-type = %q", ct)
	}
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"openapi"`) {
		t.Fatalf("json body = %s", raw)
	}

	res, err = client.Get(srv.URL + PathUI)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ui status = %d", res.StatusCode)
	}
	html, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(html)
	if !strings.Contains(page, PathJSON) {
		t.Fatalf("ui missing spec url: %s", page)
	}
	for _, vin := range []string{"1HGCM82633", "3N1AB7AP1D"} {
		if strings.Contains(page, vin) {
			t.Fatalf("ui leaked VIN %s", vin)
		}
	}
}

func TestMountDoesNotShadowSiblingRoutes(t *testing.T) {
	inner := http.NewServeMux()
	inner.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux := http.NewServeMux()
	Mount(mux, stubSpec{`{"openapi":"3.1.0"}`})
	mux.Handle("/", inner)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	res, err := srv.Client().Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", res.StatusCode)
	}
	res, err = srv.Client().Get(srv.URL + PathJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("openapi status = %d", res.StatusCode)
	}
}

func TestMountNilMuxOrEmptySpec(t *testing.T) {
	Mount(nil, stubSpec{`{}`})
	mux := http.NewServeMux()
	Mount(mux, nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, PathJSON, nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("empty spec status = %d, want 404", res.Code)
	}
}
