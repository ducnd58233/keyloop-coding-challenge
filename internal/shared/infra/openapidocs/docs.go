// Package openapidocs serves the swag-generated contract. Pass the package's
// SwaggerInfo (api/<service>/http/docs). Do not re-embed swagger.json.
package openapidocs

import (
	"fmt"
	"net/http"
)

const (
	// PathJSON is the live OpenAPI document.
	PathJSON = "/openapi.json"
	// PathUI is the Swagger UI page that loads PathJSON.
	PathUI = "/docs"

	contentJSON = "application/json"
	contentHTML = "text/html; charset=utf-8"
)

// Spec is the swag document already registered in generated docs.go.
type Spec interface {
	ReadDoc() string
}

// Mount registers JSON and a Swagger UI page from spec. Paths are constants
// so logs never include a VIN.
func Mount(mux *http.ServeMux, spec Spec) {
	if mux == nil || spec == nil {
		return
	}
	doc := spec.ReadDoc()
	if doc == "" {
		return
	}
	body := []byte(doc)
	mux.HandleFunc("GET "+PathJSON, serveBytes(contentJSON, body))
	mux.HandleFunc("GET "+PathUI, serveUI())
}

func serveBytes(contentType string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func serveUI() http.HandlerFunc {
	page := []byte(fmt.Sprintf(uiHTML, PathJSON))
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentHTML)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(page)
	}
}

// Spec URL is PathJSON. UI assets come from a CDN; /openapi.json still works offline.
const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>API docs</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: %q,
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis],
    layout: "BaseLayout"
  });
};
</script>
</body>
</html>
`
