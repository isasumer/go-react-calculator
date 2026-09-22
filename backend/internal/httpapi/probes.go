package httpapi

import "net/http"

// noStore keeps probe answers out of every cache: a stale readiness answer
// would keep a load balancer sending traffic to a draining process.
const noStore = "no-store"

// healthz is the liveness probe: it answers 200 for as long as the process
// serves, and says nothing about dependencies. A failing liveness probe means
// "restart me", so it must not fail for a reason a restart cannot fix.
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", noStore)
	h.writeJSON(w, r, http.StatusOK, StatusResponse{Status: "ok"})
}

// readyz is the readiness probe: 200 while the server accepts work, 503 with
// code NOT_READY once shutdown has begun. The lifecycle flips the flag before
// it stops the listener, so a load balancer takes this instance out of
// rotation while it can still finish the requests it has.
func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", noStore)
	if !h.ready() {
		Write(w, r, NewProblem(CodeNotReady, "the server is shutting down and is not accepting new requests"))
		return
	}
	h.writeJSON(w, r, http.StatusOK, StatusResponse{Status: "ready"})
}

// version reports the build identity of the running binary, so a deployed
// instance can be matched to a commit.
func (h *Handler) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", noStore)
	h.writeJSON(w, r, http.StatusOK, VersionResponse{
		Version:   h.build.Version,
		Commit:    h.build.Commit,
		BuildDate: h.build.BuildDate,
		GoVersion: h.build.GoVersion,
	})
}

// exposeMetrics is GET /metrics: the Prometheus exposition of the registry
// the composition root built, on the same listener as the API. It is
// registered only when [WithMetrics] supplied one.
//
// no-store for the same reason as the probes, and a sharper one: the body is
// a snapshot of this process at this instant. A cached copy would make a
// scraper compute rates from counters that never moved, which looks exactly
// like a healthy service doing nothing.
//
// Exposing it on the API's own port means anyone who can reach the API can
// read the metric names and their labels. Nothing here is a secret, and the
// ingress only publishes /api/ (ADR-0009); a separate admin listener is
// follow-up #38.
func (h *Handler) exposeMetrics() http.Handler {
	exposition := h.metrics.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", noStore)
		exposition.ServeHTTP(w, r)
	})
}
