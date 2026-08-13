// Package api is the documents HTTP surface (FR1). Status selection follows §5.4.
package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/dto"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/common"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

const (
	routeTemplate = "/api/v1/vehicles/{vin}/documents"
	maxHeaderLen  = 128
)

// DocumentsQuery is declared here so api does not import usecases by name (R1).
type DocumentsQuery interface {
	Documents(ctx context.Context, vin string) (domain.AggregateResult, error)
}

// AccessLog records FR8 without importing the audit module (R1).
type AccessLog interface {
	Record(ctx context.Context, vin, actorID, requestID string, result domain.AggregateResult, err error, latency time.Duration)
}

// Handler serves GET /api/v1/vehicles/{vin}/documents.
type Handler struct {
	query DocumentsQuery
	audit AccessLog
	log   observability.Logger
}

// New does not read the environment; timeouts live on the use case.
func New(query DocumentsQuery, audit AccessLog, log observability.Logger) *Handler {
	return &Handler{query: query, audit: audit, log: log}
}

// Register uses a constant route template so logs never contain a VIN.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+routeTemplate, h.list)
}

// list godoc
// @Summary List documents for a VIN
// @Param vin path string true "Vehicle identification number" example(1HGCM82633)
// @Param X-Actor-Id header string false "Actor recorded on the audit trail"
// @Param X-Request-Id header string false "Caller correlation id"
// @Success 200 {object} dto.DocumentsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /vehicles/{vin}/documents [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	vin := r.PathValue("vin")
	start := time.Now()
	result, err := h.query.Documents(r.Context(), vin)
	latency := time.Since(start)
	reqID := clipHeader(httpserver.RequestIDFrom(r.Context()))
	actor := clipHeader(r.Header.Get("X-Actor-Id"))

	status := http.StatusOK
	switch {
	case errors.Is(err, domain.ErrInvalidVIN):
		status = http.StatusBadRequest
	case err != nil:
		status = http.StatusServiceUnavailable
	}

	h.trace(r.Context(), vin, status)
	switch {
	case errors.Is(err, domain.ErrInvalidVIN):
		httpserver.JSON(w, status, dto.InvalidVIN())
	case err != nil:
		// Unexpected errors stay 503 with a safe message, not driver text.
		httpserver.JSON(w, status, dto.Unavailable())
	default:
		httpserver.JSON(w, status, dto.FromAggregate(vin, reqID, result))
	}
	// Write first so a slow FR8 insert cannot blow the NFR4 budget.
	if h.audit != nil {
		h.audit.Record(r.Context(), vin, actor, reqID, result, err, latency)
	}
}

func (h *Handler) trace(ctx context.Context, vin string, status int) {
	if h.log == nil {
		return
	}
	h.log.InfoContext(ctx, "documents lookup",
		"route", routeTemplate,
		"vin_suffix", common.VINSuffix(vin),
		"status", status,
	)
}

func clipHeader(v string) string {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) <= maxHeaderLen {
		return v
	}
	runes := []rune(v)
	return string(runes[:maxHeaderLen])
}
