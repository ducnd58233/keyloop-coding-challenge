package http

import (
	"net/url"
	"strings"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

// Closed A4 type vocabulary. Unrecognised upstream values become OTHER (FR5).
var knownTypes = map[string]struct{}{
	"INVOICE":      {},
	"FINANCE":      {},
	"REGISTRATION": {},
	"CONTRACT":     {},
	"QUOTE":        {},
	"WORK_ORDER":   {},
	"INSPECTION":   {},
	"WARRANTY":     {},
	"RECALL":       {},
	"MOT":          {},
}

// MapType uppercases and underscores the upstream label. Unknown stays visible as OTHER.
func MapType(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	if s == "" {
		return "OTHER"
	}
	if _, ok := knownTypes[s]; ok {
		return s
	}
	return "OTHER"
}

func resolveURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	parsed, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	root, err := url.Parse(strings.TrimRight(base, "/") + "/")
	if err != nil {
		return ""
	}
	return root.ResolveReference(parsed).String()
}

func namespaced(source domain.SourceName, localID string) string {
	return domain.NamespacedID(source, strings.TrimSpace(localID))
}
