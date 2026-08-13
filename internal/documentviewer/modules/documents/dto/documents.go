// Package dto holds documents wire types. JSON tags stay out of domain (R2).
package dto

import (
	"strings"
	"time"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
)

// ErrorResponse is the viewer error envelope. HTTP status is the code;
// details is only set for field validation (400).
type ErrorResponse struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Envelope copy is fixed so 400/503 stay stable for clients and OpenAPI.
const (
	MsgInvalidVIN   = "VIN format is invalid"
	MsgUnavailable  = "all document sources are unavailable"
	FieldVIN        = "vin"
	DetailVINFormat = "must be exactly 10 alphanumeric characters"
)

// InvalidVIN is 400. Details name the field; they never echo the submitted VIN.
func InvalidVIN() ErrorResponse {
	return ErrorResponse{
		Message: MsgInvalidVIN,
		Details: map[string]string{FieldVIN: DetailVINFormat},
	}
}

// Unavailable is 503. No details: this is not a validation failure.
func Unavailable() ErrorResponse {
	return ErrorResponse{Message: MsgUnavailable}
}

// Document is exactly the six A4 fields in snake_case.
type Document struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	IssuedAt string `json:"issued_at"`
	URL      string `json:"url"`
}

// SourceError is FR7 per-source failure detail. Codes only, no hostnames.
type SourceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Source is one FR7 sources[] entry. Failed upstreams wire as UNAVAILABLE.
type Source struct {
	Name          string       `json:"name"`
	Status        string       `json:"status"`
	LatencyMs     int          `json:"latency_ms"`
	DocumentCount int          `json:"document_count"`
	Error         *SourceError `json:"error,omitempty"`
}

// DocumentsResponse is SYSTEM_DESIGN §5.2.
type DocumentsResponse struct {
	VIN             string     `json:"vin"`
	Documents       []Document `json:"documents"`
	Sources         []Source   `json:"sources"`
	Partial         bool       `json:"partial"`
	ServedFromCache bool       `json:"served_from_cache"`
	Stale           bool       `json:"stale"`
	RequestID       string     `json:"request_id"`
}

// FromAggregate maps domain result to the wire shape. Empty lists are [] not null.
func FromAggregate(vin, requestID string, r domain.AggregateResult) DocumentsResponse {
	docs := make([]Document, 0, len(r.Documents))
	for _, d := range r.Documents {
		docs = append(docs, Document{
			ID:       d.ID,
			Source:   string(d.Source),
			Type:     d.Type,
			Title:    d.Title,
			IssuedAt: d.IssuedAt.UTC().Format(time.RFC3339),
			URL:      d.URL,
		})
	}
	sources := make([]Source, 0, len(r.Sources))
	for _, s := range r.Sources {
		sources = append(sources, fromSource(s))
	}
	return DocumentsResponse{
		VIN:             vin,
		Documents:       docs,
		Sources:         sources,
		Partial:         r.Partial,
		ServedFromCache: r.ServedFromCache,
		Stale:           r.Stale,
		RequestID:       requestID,
	}
}

func fromSource(s domain.SourceReport) Source {
	out := Source{
		Name:          string(s.Name),
		Status:        "OK",
		LatencyMs:     s.LatencyMs,
		DocumentCount: s.DocumentCount,
	}
	if s.Status == domain.SourceStatusOK {
		return out
	}
	out.Status = "UNAVAILABLE"
	out.Error = &SourceError{
		Code:    s.ErrorCode,
		Message: sourceMessage(s.Name, s.ErrorCode),
	}
	return out
}

func sourceMessage(name domain.SourceName, code string) string {
	src := strings.ToLower(string(name))
	if src == "" {
		src = "source"
	}
	switch code {
	case domain.CodeUpstreamTimeout:
		return src + " fetch exceeded per-source budget"
	default:
		return src + " fetch failed"
	}
}
