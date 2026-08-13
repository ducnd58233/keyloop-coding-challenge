package httpserver

import (
	"encoding/json"
	"net/http"
)

// JSON is the only encoder so content-type and encoding stay in one place.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
