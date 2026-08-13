package domain

import (
	"sort"
	"strings"
)

// NamespacedID prevents cross-source ID collisions (DD-8).
func NamespacedID(source SourceName, localID string) string {
	return strings.ToLower(string(source)) + ":" + localID
}

// Merge sorts by issued_at descending, then source name, then id (DD-7).
func Merge(docs []Document) []Document {
	out := append([]Document(nil), docs...)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].IssuedAt.Equal(out[j].IssuedAt) {
			return out[i].IssuedAt.After(out[j].IssuedAt)
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].ID < out[j].ID
	})
	return out
}
