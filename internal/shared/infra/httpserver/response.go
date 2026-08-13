package httpserver

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as the response body with the given status code. Handlers
// across modules use this instead of encoding inline, so the content type
// and encoding behaviour are set in exactly one place.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
