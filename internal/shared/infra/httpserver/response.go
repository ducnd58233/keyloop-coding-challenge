package httpserver

import (
	"encoding/json"
	"net/http"
)

const contentTypeJSON = "application/json"

// JSON is the only encoder so content-type and encoding stay in one place.
// T is a concrete DTO; encoding any would hide marshal failures.
func JSON[T any](w http.ResponseWriter, status int, v T) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(b)
}
