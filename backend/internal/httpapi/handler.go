package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/isasumer/go-react-calculator/backend/internal/calc"
	"github.com/isasumer/go-react-calculator/backend/internal/observability"
)

// unknownOperation is the calc_operations_total operation label for a
// request that never named an operation the registry knows.
const unknownOperation calc.Operation = "unknown"

// Handler serves the calculator API. Build it with [NewHandler].
type Handler struct {
	calc    *calc.Registry
	log     *slog.Logger
	ready   func() bool
	build   observability.BuildInfo
	metrics *observability.Metrics
}

// An Option overrides a Handler default. Options exist for what the
// composition root owns — the process lifecycle and the build identity — so
// the handler itself stays free of globals.
type Option func(*Handler)

// WithReadiness makes GET /readyz report ready only while isReady returns
// true. The flag is owned by the server lifecycle, which flips it off before
// draining. A nil isReady is ignored; the default is always ready.
func WithReadiness(isReady func() bool) Option {
	return func(h *Handler) {
		if isReady != nil {
			h.ready = isReady
		}
	}
}

// WithBuildInfo sets what GET /version reports. The default is
// [observability.Build].
func WithBuildInfo(b observability.BuildInfo) Option {
	return func(h *Handler) { h.build = b }
}

// WithMetrics records every calculation in m and mounts m's exposition at
// GET /metrics. Without it the handler serves no /metrics route and counts
// nothing, which is what a test that only cares about the API wants.
func WithMetrics(m *observability.Metrics) Option {
	return func(h *Handler) { h.metrics = m }
}

// NewHandler returns a Handler evaluating with reg. A nil log discards
// output. Without options the handler is always ready and reports this
// binary's own build information.
func NewHandler(reg *calc.Registry, log *slog.Logger, opts ...Option) *Handler {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	h := &Handler{
		calc:  reg,
		log:   log,
		ready: func() bool { return true },
		build: observability.Build(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Routes returns the API router. Wrong methods on a known path get a 405
// problem with an Allow header; unknown paths get a 404 problem.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	// A pattern with a method is more specific than the same path without
	// one, so the method-less patterns only see the methods not listed.
	mux.HandleFunc("POST /api/v1/calculate", h.calculate)
	mux.HandleFunc("/api/v1/calculate", h.methodNotAllowed(http.MethodPost))
	mux.HandleFunc("GET /api/v1/operations", h.operations)
	mux.HandleFunc("/api/v1/operations", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	mux.HandleFunc("GET /api/v1/openapi.yaml", h.openapiSpec)
	mux.HandleFunc("/api/v1/openapi.yaml", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	// Operational endpoints, outside the versioned API: they are for the
	// platform (probes, deploy tooling), not for API clients.
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("/healthz", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.HandleFunc("/readyz", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	mux.HandleFunc("GET /version", h.version)
	mux.HandleFunc("/version", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	if h.metrics != nil {
		mux.Handle("GET /metrics", h.exposeMetrics())
		mux.HandleFunc("/metrics", h.methodNotAllowed(http.MethodGet, http.MethodHead))
	}
	mux.HandleFunc("/", h.notFound)
	return mux
}

// calculate answers the one endpoint that does arithmetic, and records the
// outcome. The work is in evaluate so there is a single place where an
// attempt is finished, and therefore a single record call.
func (h *Handler) calculate(w http.ResponseWriter, r *http.Request) {
	op, resp, problem := h.evaluate(w, r)
	h.record(op, problem)
	if problem != nil {
		Write(w, r, problem)
		return
	}
	h.writeJSON(w, r, http.StatusOK, resp)
}

// evaluate runs the request through decode, validate and the calculator. It
// returns the operation the request asked for — empty until validation has
// confirmed the registry knows it — and either a response or the problem to
// send instead.
func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request) (calc.Operation, CalculateResponse, *Problem) {
	req, err := decodeJSON[CalculateRequest](w, r)
	if err != nil {
		return "", CalculateResponse{}, h.mapError(err)
	}
	spec, err := h.validate(req)
	if err != nil {
		return "", CalculateResponse{}, h.mapError(err)
	}
	result, err := h.calc.Evaluate(spec.Name, *req.A, req.B)
	if err != nil {
		return spec.Name, CalculateResponse{}, h.mapError(&evalError{op: spec.Name, err: err})
	}
	return spec.Name, CalculateResponse{
		Operation: string(spec.Name),
		A:         *req.A,
		B:         req.B,
		Result:    result,
	}, nil
}

// record counts one attempted calculation: outcome "ok", or the stable code
// of the problem it failed with.
//
// An operation the registry never recognized is counted as
// [unknownOperation] rather than as whatever string the client sent. The
// label would otherwise be attacker-controlled, and one request per made-up
// operation name is one time series per made-up operation name.
func (h *Handler) record(op calc.Operation, problem *Problem) {
	if h.metrics == nil {
		return
	}
	outcome := observability.OutcomeOK
	if problem != nil {
		outcome = string(problem.Code)
	}
	if op == "" {
		op = unknownOperation
	}
	h.metrics.ObserveCalculation(string(op), outcome)
}

func (h *Handler) operations(w http.ResponseWriter, r *http.Request) {
	specs := h.calc.List()
	resp := OperationsResponse{Operations: make([]OperationInfo, len(specs))}
	for i, s := range specs {
		resp.Operations[i] = OperationInfo{Name: string(s.Name), Symbol: s.Symbol, Arity: s.Arity}
	}
	h.writeJSON(w, r, http.StatusOK, resp)
}

func (h *Handler) methodNotAllowed(allowed ...string) http.HandlerFunc {
	allow := strings.Join(allowed, ", ")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		Write(w, r, NewProblem(CodeMethodNotAllowed, "method "+r.Method+" is not allowed; use "+allow))
	}
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	Write(w, r, NewProblem(CodeNotFound, "no route for "+r.Method+" "+r.URL.Path))
}
