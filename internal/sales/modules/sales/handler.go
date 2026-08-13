package sales

import (
	"log/slog"
	"net/http"

	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockfault"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/mockseed"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Options keeps Generate off when Down, or unknown VINs look healthy during outage.
type Options struct {
	Fault    mockfault.Config
	Generate bool
	IntN     func(n int) int
	Log      observability.Logger
}

// New is the Sales mock HTTP surface (A5).
func New(opt Options) http.Handler {
	s := server{fault: opt.Fault, generate: opt.Generate, intN: opt.IntN, log: opt.Log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /sales/v1/documents", s.list)
	return mux
}

type server struct {
	fault    mockfault.Config
	generate bool
	intN     func(n int) int
	log      observability.Logger
}

// healthz godoc
// @Summary Liveness probe
// @Success 200 {string} string
// @Failure 503 {string} string
// @Router /healthz [get]
func (s server) healthz(w http.ResponseWriter, _ *http.Request) {
	if s.fault.Down {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// list godoc
// @Summary List sales documents for a VIN
// @Param vin query string true "Vehicle identification number"
// @Success 200 {object} listResponse
// @Failure 500 {string} string
// @Failure 503 {string} string
// @Router /sales/v1/documents [get]
func (s server) list(w http.ResponseWriter, r *http.Request) {
	vin := r.URL.Query().Get("vin")
	fault := s.fault.Apply(r.Context(), w)
	if fault.Stop {
		s.traceFault(r, vin, fault)
		return
	}
	records, ok := catalog[vin]
	generated := false
	if !ok {
		if s.generate {
			out, err := generateRecords(vin, s.intN)
			if err != nil {
				s.traceError(r, vin, "generate", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			records = out
			generated = true
		} else {
			records = []record{}
		}
	}
	s.traceOK(r, vin, fault, len(records), generated)
	httpserver.JSON(w, http.StatusOK, listResponse{VIN: vin, Records: records})
}

func (s server) baseAttrs(r *http.Request, vin string, fault mockfault.Result) []any {
	return []any{
		slog.String("request_id", httpserver.RequestIDFrom(r.Context())),
		slog.String("method", r.Method),
		slog.String("route", "/sales/v1/documents"),
		slog.String("vin_suffix", mockseed.Suffix(vin)),
		slog.String("fault", string(fault.Kind)),
		slog.Duration("latency", fault.Latency),
	}
}

func (s server) traceOK(r *http.Request, vin string, fault mockfault.Result, records int, generated bool) {
	if s.log == nil {
		return
	}
	s.log.Info("sales response", append(s.baseAttrs(r, vin, fault),
		slog.Int("records", records),
		slog.Bool("generated", generated),
		slog.Int("status", http.StatusOK),
	)...)
}

func (s server) traceFault(r *http.Request, vin string, fault mockfault.Result) {
	if s.log == nil {
		return
	}
	status := 0
	switch fault.Kind {
	case mockfault.KindDown:
		status = http.StatusServiceUnavailable
	case mockfault.KindError:
		status = http.StatusInternalServerError
	case mockfault.KindOK, mockfault.KindTimeout, mockfault.KindLatency:
		status = 0
	}
	s.log.Warn("sales fault", append(s.baseAttrs(r, vin, fault), slog.Int("status", status))...)
}

func (s server) traceError(r *http.Request, vin, op string, err error) {
	if s.log == nil {
		return
	}
	s.log.Error("sales error",
		slog.String("request_id", httpserver.RequestIDFrom(r.Context())),
		slog.String("vin_suffix", mockseed.Suffix(vin)),
		slog.String("op", op),
		slog.String("error", err.Error()),
	)
}
