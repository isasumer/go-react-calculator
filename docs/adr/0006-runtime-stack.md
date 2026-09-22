# ADR-0006: Build the service on the standard library, with the Prometheus client as the only runtime dependency

- **Status:** Accepted
- **Date:** 2026-09-23
- **Ticket:** #9

## Context
The assignment is a calculator API judged on idiomatic Go, clarity and maintainability, and it is read by
someone who has to understand the whole repository quickly. Every dependency added is something that
reader has to trust, that CI has to keep patched, and that a reviewer cannot skim. Go 1.22 moved method
and wildcard patterns into `net/http.ServeMux`, and Go 1.21 put structured logging in `log/slog`, so the
two things a service normally imports a framework for are now in the standard library.

What the standard library does not have is a metrics format. An operator asking "is it up, how fast is it,
what is failing, and which operations do people actually use" needs numbers that aggregate, not log lines
that have to be parsed. Prometheus is the de-facto scrape format; `github.com/prometheus/client_golang` is
its reference implementation, and everything that ingests metrics — Prometheus itself, Grafana Alloy,
Datadog, the OpenTelemetry collector — reads what it writes.

## Decision
- **Router:** `net/http.ServeMux` with Go 1.22 method patterns (`POST /api/v1/calculate`). No chi, gin,
  echo or fiber. The middleware chain is `func(http.Handler) http.Handler`, composed in one list in the
  composition root (`internal/middleware.Chain`).
- **Logging:** `log/slog`, JSON by default, built once in `cmd/server` from the resolved config and
  injected. No logrus/zap/zerolog.
- **Metrics:** `github.com/prometheus/client_golang`. It is the only non-`golang.org/x` module the
  service imports at runtime; `golang.org/x/time/rate` (the rate limiter) is the other dependency and is
  a Go project module. Five families, defined once in `internal/observability`:
  `http_requests_total{method,route,status}`, `http_request_duration_seconds{method,route}`,
  `http_in_flight_requests`, `calc_operations_total{operation,outcome}` and `build_info{version,commit}`.
- **No global registry.** `prometheus.NewRegistry()` plus the Go and process collectors, created in the
  composition root and injected. Never `prometheus.DefaultRegisterer`: a global registry is shared with
  every library that ever imports the client, cannot be reset between tests, and turns a duplicate
  registration into a startup panic in someone else's code.
- **Route, not path.** The `route` label is the pattern `http.ServeMux` matched (`r.Pattern`, with the
  method stripped because it is already its own label), so a client cannot mint time series by inventing
  URLs. The metrics middleware is therefore innermost in the chain, directly around the router: it is the
  only position from which the matched pattern can be read.
- **Operation labels come from the registry**, never from the request body. An unsupported operation is
  counted as `operation="unknown"`, for the same cardinality reason.
- **One listener.** `GET /metrics` is served on the API's port, next to `/healthz`, `/readyz` and
  `/version`, with `Cache-Control: no-store`. Rate limiting skips it, as it skips the probes.

## Consequences
- A reviewer can read the entire request path in this repository: there is no router, logger or metrics
  framework whose behavior has to be taken on faith. `go.mod` stays short enough to audit in one glance,
  which also keeps `govulncheck` and Dependabot quiet.
- The middleware's position buys the `route` label and costs the responses produced above it. A CORS
  preflight and a `429` from the rate limiter are never counted; a request that `Timeout` answered `503`
  is counted with whatever status its handler produces when it eventually returns, which is not the status
  the client received. Latency is truthful in every case, and those failure modes have their own signals
  (the access log, `RATE_LIMITED` and `TIMEOUT` in the problem counts). Moving metrics outward would swap
  a known, documented gap for an unbounded label — a worse trade.
- `build_info` makes a deploy joinable onto any other series, which is how a latency change gets attributed
  to a release without a second data source.
- Metric names and labels are now a public interface. Renaming one breaks dashboards and alerts, so they
  change the way the error codes in ADR-0005 change: additively.
- Anything the standard library gains later (a metrics API, richer routing) can replace a piece of this
  without touching the rest, because each concern is one small package behind a constructor.

## Alternatives considered
- **chi / gin / echo** → the two features worth having, method patterns and path wildcards, are in
  `net/http` since 1.22. A framework would add a dependency, a second set of idioms, and a layer between
  the reader and the request.
- **zap / zerolog** → faster in benchmarks that do not resemble this service, which logs one line per
  request. `slog` is the standard, and `slog.Handler` keeps the format a deployment choice.
- **OpenTelemetry metrics (`otel` SDK + Prometheus exporter)** → the vendor-neutral answer, and where this
  would go if tracing were in scope. Today it is three modules and an exporter to express five families
  that the Prometheus client expresses directly, and the correlation that makes OTel worth its weight
  comes from traces, which this ticket explicitly excludes. Deferred to follow-up
  [#32](https://github.com/isasumer/go-react-calculator/issues/32), which adds tracing and a Grafana stack
  together; the metric *names* here are already the OTel HTTP semantic-convention names, so that migration
  is an exporter swap rather than a dashboard rewrite.
- **A separate admin listener for `/metrics` and the probes** → the right production shape, because it
  keeps the operational surface off the port the internet reaches. It is a second `http.Server`, a second
  port in config, compose, the Dockerfile and the Kubernetes manifests, and a second lifecycle to drain.
  Nothing exposed here is a secret, and the ingress publishes only `/api/` (ADR-0009), so the split is
  follow-up [#38](https://github.com/isasumer/go-react-calculator/issues/38) rather than scope in a
  Sprint 1 ticket.
- **`expvar`** → in the standard library, and read by nothing an operations team runs.
- **Pull the numbers out of the access log** (Loki, CloudWatch Insights) → works, at the cost of computing
  rates over parsed text at query time. A counter is cheaper to store, cheaper to alert on, and does not
  break when a log field is renamed.
