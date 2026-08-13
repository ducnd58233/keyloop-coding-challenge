// Package api is the documents HTTP surface (FR1). It selects status per §5.4.
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
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

const (
	routeTemplate = "/api/v1/vehicles/{vin}/documents"
	maxHeaderLen  = 128
)

// DocumentsQuery is the aggregate use case. Declared here so api does not import usecases by name.
type DocumentsQuery interface {
	Documents(ctx context.Context, vin string) (domain.AggregateResult, error)
}

// AccessLog records FR8 without importing the audit module (R1).
type AccessLog interface {
	Record(ctx context.Context, vin, actorID, requestID string, result domain.AggregateResult, err error, latency time.Duration)
}

// Handler mounts GET /api/v1/vehicles/{vin}/documents.
type Handler struct {
	query DocumentsQuery
	audit AccessLog
	log   observability.Logger
}

// New does not read the environment; timeouts live on the use case.
func New(query DocumentsQuery, audit AccessLog, log observability.Logger) *Handler {
	return &Handler{query: query, audit: audit, log: log}
}

// Register attaches document routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+routeTemplate, h.list)
}

// list godoc
// @Summary List documents for a VIN
// @Param vin path string true "Vehicle identification number"
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
	reqID := httpserver.RequestIDFrom(r.Context())
	actor := clipHeader(r.Header.Get("X-Actor-Id"))

	if h.audit != nil {
		h.audit.Record(r.Context(), vin, actor, reqID, result, err, latency)
	}

	status := http.StatusOK
	switch {
	case errors.Is(err, domain.ErrInvalidVIN):
		status = http.StatusBadRequest
		h.trace(r.Context(), vin, status)
		httpserver.JSON(w, status, dto.ErrorResponse{Error: dto.ErrorBody{Code: domain.CodeInvalidVIN}})
		return
	case errors.Is(err, domain.ErrAllSourcesUnavailable), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		status = http.StatusServiceUnavailable
		h.trace(r.Context(), vin, status)
		httpserver.JSON(w, status, dto.ErrorResponse{Error: dto.ErrorBody{Code: domain.CodeAllSourcesUnavailable}})
		return
	case err != nil:
		status = http.StatusInternalServerError
		h.trace(r.Context(), vin, status)
		httpserver.JSON(w, status, dto.ErrorResponse{Error: dto.ErrorBody{Code: "INTERNAL_ERROR"}})
		return
	default:
		h.trace(r.Context(), vin, status)
		httpserver.JSON(w, status, dto.FromAggregate(vin, reqID, result))
	}
}

func (h *Handler) trace(ctx context.Context, vin string, status int) {
	if h.log == nil {
		return
	}
	h.log.InfoContext(ctx, "documents lookup",
		"route", routeTemplate,
		"vin_suffix", vinSuffix(vin),
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

func vinSuffix(vin string) string {
	runes := []rune(vin)
	if len(runes) <= 4 {
		return ""
	}
	return string(runes[len(runes)-4:])
}
