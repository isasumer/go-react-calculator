# Backend

Go service for the calculator API. Module: `github.com/isasumer/go-react-calculator/backend`.
It serves `POST /api/v1/calculate` and `GET /api/v1/operations` (contract: [`docs/PLAN.md` §1.4](../docs/PLAN.md#14-api-contract-frozen-after-b1-02-changes-require-an-adr), errors: [`docs/errors.md`](../docs/errors.md)) plus the operational endpoints below. The machine-readable contract is
[`api/openapi.yaml`](api/openapi.yaml), served at `GET /api/v1/openapi.yaml`. Every request travels through the middleware chain described below and is counted in the Prometheus families `GET /metrics` exposes.

## Layout

```
api/                     openapi.yaml: the API contract, embedded and served (B1-06)
cmd/server/              composition root; main() calls run(ctx, args, getenv, stdout)
internal/calc/           pure domain (B1-01)
internal/httpapi/        HTTP transport, problem+json (B1-02)
internal/middleware/     middleware chain: recovery, request ID, logging, timeout, security, CORS, rate limit, metrics
internal/config/         env → typed, validated Config with defaults
internal/observability/  build information; Prometheus registry and metric families
scripts/                 coverage-check.sh
tools/                   separate Go module that pins dev tool versions
```

## Tools

The dev tools are pinned with `tool` directives in `tools/go.mod` (and `tools/go.sum`). They live in
their own module, so none of their dependencies end up in the service's `go.mod`.

| Tool | Version | Used by |
|---|---|---|
| golangci-lint | v2.13.2 | `make lint` |
| gofumpt | v0.12.0 | `make fmt`, `make fmt-check` |
| govulncheck | v1.8.0 | `make vuln` |

Install them with:

```sh
make tools        # builds the pinned tools into ./bin (gitignored)
```

Every target that needs a tool also builds it on first use. To bump a version:
`cd tools && go get -tool <module>@<version> && go mod tidy`, then update the table above.

Requirements: Go ≥ 1.27 on `PATH`.

## Targets

| Target | What it does |
|---|---|
| `make check` | fmt-check → lint → vet → test → coverage-check (the gate CI and `make check` at the root run) |
| `make fmt` / `fmt-check` | format with gofumpt / fail if anything is unformatted |
| `make lint` | golangci-lint with `.golangci.yml` |
| `make vet` | `go vet ./...` |
| `make test` | `go test -race -shuffle=on -coverprofile=coverage.out ./...` |
| `make cover` | per-function coverage summary |
| `make coverage-check` | fails when total coverage is below 85 % |
| `make vuln` | govulncheck |
| `make build` | static binary in `bin/server`, with `Version`, `Commit` and `BuildDate` stamped into `internal/observability` via `-ldflags -X` |
| `make run` | runs the server (`PORT=9000 make run` to change the port) |

`./bin/server -version` prints the stamped build identity and exits 0. Without the stamps (`go run`),
the values fall back to the VCS information the toolchain embeds, then to `dev`.

## Configuration

The process is configured entirely by the environment (12-factor); there are no config files and no
flags other than `-version`. Every variable is optional: unset — or set to an empty/blank string —
means the default. An unparsable value is fatal at startup and the message names the variable and the
value received, e.g. `config: PORT="eight thousand": must be an integer`. All bad values are reported
at once, so one restart shows every mistake.

| Variable | Default | Meaning |
|---|---|---|
| `HOST` | `0.0.0.0` | Interface to bind. `127.0.0.1` keeps the server off the network. |
| `PORT` | `8081` | TCP port, `0`–`65535`. `0` asks the kernel for a free port; the bound port is in the startup log line. |
| `LOG_LEVEL` | `info` | Minimum slog level: `debug`, `info`, `warn` or `error`. |
| `LOG_FORMAT` | `json` | slog handler: `json` for shipping, `text` for reading locally. |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | Comma-separated allowlist for the CORS middleware; spaces around items are trimmed and empty items dropped. Empty means no cross-origin browser access at all — not even a `Vary` header — because production serves the frontend same-origin (ADR-0009). |
| `READ_HEADER_TIMEOUT` | `5s` | `http.Server.ReadHeaderTimeout`: how long a client may take to send the request headers. |
| `READ_TIMEOUT` | `10s` | `http.Server.ReadTimeout`: headers plus body. |
| `WRITE_TIMEOUT` | `10s` | `http.Server.WriteTimeout`: how long a response may take to write. |
| `IDLE_TIMEOUT` | `60s` | `http.Server.IdleTimeout`: how long a keep-alive connection may sit idle. |
| `SHUTDOWN_TIMEOUT` | `15s` | Budget for draining in-flight requests after the pre-stop delay. Must be greater than zero. |
| `PRE_STOP_DELAY` | `0s` | Pause between flipping `/readyz` to 503 and closing the listener. Keep it `0` locally; in Kubernetes set it to about twice the readiness probe period (e.g. `5s`) so the endpoint is removed from every load balancer *before* the process stops accepting connections. |
| `REQUEST_TIMEOUT` | `5s` | Per-request budget. When it runs out the handler's context is canceled and the client gets a `503` problem with `code: TIMEOUT`. `0` disables the middleware. |
| `RATE_LIMIT_RPS` | `20` | Sustained requests per second per client, per process. Minimum `1`. |
| `RATE_LIMIT_BURST` | `40` | How many requests a client may make back to back before that rate applies. Minimum `1`. |
| `MAX_BODY_BYTES` | `4096` | Maximum accepted request body; larger bodies get `413 PAYLOAD_TOO_LARGE`. |
| `TRUST_PROXY_HEADERS` | `false` | Whether the rate limiter may key on the first `X-Forwarded-For` entry instead of the peer address. Only enable behind a proxy that overwrites the header; with nothing rewriting it, a client can mint a fresh bucket per request. (An incoming `X-Request-ID` is honoured regardless, when it is syntactically valid — see below.) |

The timeouts accept any Go duration (`750ms`, `5s`, `2m`); `0` disables the `http.Server` ones.
`MAX_BODY_BYTES` is parsed and logged but not yet enforced — the body limit is still the 4096-byte
constant in `internal/httpapi` (follow-up [#62](https://github.com/isasumer/go-react-calculator/issues/62)).

## Middleware

`cmd/server` assembles the chain with `middleware.Chain(router, …)`; the first middleware listed is the
outermost, so the list reads in the order a request travels (`docs/PLAN.md` §1.2). Each layer is one file
in `internal/middleware` with its own test.

| # | Layer | What it does |
|---|---|---|
| 1 | `Recover` | Catches a panic anywhere below it, logs it with its stack at `error`, and answers `500` `code: INTERNAL`. The panic value never reaches the client, and the process keeps serving. |
| 2 | `RequestID` | Reuses an incoming `X-Request-ID` matching `^[A-Za-z0-9_-]{1,64}$`, otherwise generates 16 bytes from `crypto/rand` as unpadded base32. Sets it on the response before the handler runs and puts it in the request context, so log lines and problem documents quote the same ID. |
| 3 | `Logger` | One `slog` line per request: `request_id`, `method`, `path`, `route`, `status`, `bytes`, `duration_ms`, `remote_ip`, `user_agent`. Level follows the status (5xx `error`, 4xx `warn`, else `info`); `/healthz`, `/readyz` and `/metrics` drop to `debug` when they answer normally. |
| 4 | `Timeout` | Runs the handler with a `REQUEST_TIMEOUT` context and answers `503` `code: TIMEOUT` when it runs out, locking the late handler out of the response. Not `http.TimeoutHandler`: that one writes an HTML body. |
| 5 | `SecurityHeaders` | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, and `Cache-Control: no-store` on `/api/` routes. |
| 6 | `CORS` | Allowlist from `CORS_ALLOWED_ORIGINS`, echoing the origin and never `*`; preflight answered `204` with `Allow-Methods`, `Allow-Headers` and `Max-Age: 600`; `Vary: Origin` on everything it looks at. An empty allowlist disables it completely. |
| 7 | `RateLimit` | Per-client token bucket (`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`, `golang.org/x/time/rate`) with TTL eviction of idle buckets. Over the limit: `429` `code: RATE_LIMITED` with `Retry-After`. Skips the operational endpoints. |
| 8 | `Metrics` | Counts every request into `http_requests_total`, times it into `http_request_duration_seconds` and holds it in `http_in_flight_requests`. Innermost, directly around the router, because that is the only place the matched route pattern can be read. Skips nothing below it. |

Two consequences of that order are worth knowing. A panicking request produces no access log line — it
unwinds past the logger before there is a status to report — and is logged once by `Recover` instead, under
the same request ID. And a timed-out response carries the request ID but not the headers layers 5 and 6
staged, because the handler goroutine is still running and its header map cannot be read safely;
`http.TimeoutHandler` drops them for the same reason.

## API contract

[`api/openapi.yaml`](api/openapi.yaml) describes the whole HTTP surface in OpenAPI 3.1: both `/api/v1`
endpoints, the operational ones below, the `Problem` document with the full `code` enum from
[`docs/errors.md`](../docs/errors.md), and one example per error — the same bytes as the golden files in
`internal/httpapi/testdata`, which is what `TestContractExamplesMatchGoldenFiles` enforces.

The file is embedded with `//go:embed` and served at `GET /api/v1/openapi.yaml`
(`application/yaml`, `Cache-Control: no-store`), so a running instance hands out the contract the binary
was built from and there is no second copy to drift.

It is a test, not a document that happens to sit in the repository:
`internal/httpapi/openapi_test.go` loads it, validates it with
[kin-openapi](https://github.com/getkin/kin-openapi) (a test-only dependency) and, through
`assertMatchesSpec`, validates **every response the handler table tests produce** against it. A status
the contract does not document, a `code` that does not belong to its status, a body that does not match
its schema — each fails the test. In CI, `npx @stoplight/spectral-cli lint` (ruleset
[`.spectral.yaml`](../.spectral.yaml)) lints the document itself in the backend workflow's lint job.

View it as documentation with Redoc (from the repository root):

```bash
docker run --rm -p 8080:80 -v "$PWD/backend/api:/usr/share/nginx/html/spec:ro" \
  -e SPEC_URL=spec/openapi.yaml redocly/redoc   # then open http://localhost:8080
```

or with Swagger UI, swapping the image for
`-e SWAGGER_JSON=/spec/openapi.yaml -v "$PWD/backend/api:/spec:ro" -p 8080:8080 swaggerapi/swagger-ui`.
Against a running server, point either at `http://localhost:8081/api/v1/openapi.yaml` instead.

## Operational endpoints

These sit outside `/api/v1`: they are for the platform, not for API clients. All four answer
`GET` (and `HEAD`), send `Cache-Control: no-store`, and return a `405` problem with an `Allow`
header for any other method.

| Endpoint | Answers |
|---|---|
| `GET /healthz` | Liveness: `200 {"status":"ok"}` for as long as the process serves, including while it drains — a draining process must not be restarted. |
| `GET /readyz` | Readiness: `200 {"status":"ready"}` while the server accepts work; `503` problem+json with `code: NOT_READY` from the moment shutdown starts, before the listener closes. |
| `GET /version` | `200 {"version","commit","buildDate","goVersion"}` for the running binary. |
| `GET /metrics` | Prometheus text exposition of this process; see [Metrics](#metrics). |

## Metrics

The registry is a private `prometheus.NewRegistry()` built in `cmd/server` and injected — never
`prometheus.DefaultRegisterer` — carrying the Go runtime and process collectors (`go_*`, `process_*`)
alongside the five families below. `GET /metrics` serves it on the API's own port; a separate admin
listener is follow-up [#38](https://github.com/isasumer/go-react-calculator/issues/38). Rationale and
trade-offs: [ADR-0006](../docs/adr/0006-runtime-stack.md).

| Family | Type | Labels | What to alert on |
|---|---|---|---|
| `http_requests_total` | counter | `method`, `route`, `status` | `sum by (route) (rate(…{status=~"5.."}[5m])) / sum by (route) (rate(…[5m]))` above 1 % for 5 min — the service itself is failing; 4xx is the client's problem and belongs on a dashboard, not a pager. |
| `http_request_duration_seconds` | histogram | `method`, `route` | `histogram_quantile(0.99, sum by (le, route) (rate(…_bucket[5m])))` above 0.25 s — this service does one floating-point operation, so a p99 out of the low buckets means contention or a stuck dependency, not load. |
| `http_in_flight_requests` | gauge | — | A level that stays high while `rate(http_requests_total[5m])` is flat: requests are arriving and not leaving, which is the shape of a wedged handler well before the timeout budget shows it. |
| `calc_operations_total` | counter | `operation`, `outcome` | Any `rate(…{outcome="INTERNAL"}[5m]) > 0` — a bug in the calculator. The domain outcomes (`DIVISION_BY_ZERO`, `DOMAIN_ERROR`, `RESULT_NOT_FINITE`, `VALIDATION_FAILED`) are users being users: chart their share, never page on it. |
| `build_info` | gauge, always 1 | `version`, `commit` | Nothing. `changes(build_info[1h])` annotates deploys on every other graph, which is how a latency step gets attributed to a release. |

Two label rules keep the cardinality bounded by this repository rather than by what clients send:

- `route` is the pattern the router matched (`/api/v1/calculate`), never the request target
  (`/api/v1/calculate?x=1`) — with the method stripped, since `method` is its own label. A request the
  router did not match is `route="unmatched"`.
- `operation` is a name from the `calc` registry. An unsupported operation is counted as
  `operation="unknown"`, not as the string the client sent.

What the middleware's position costs is written down rather than papered over: a CORS preflight and a
`429` from the rate limiter never reach it, and a request that `Timeout` answered `503` is counted with
the status its handler produces when it finally returns. The access log carries all three.

## Lifecycle

`main` only owns the process: it turns SIGINT/SIGTERM into a canceled context and maps an error to
exit 1. Everything else is `run(ctx, args, getenv, stdout)`, which the tests drive exactly as `main`
does. `run` listens before it serves, so a bind failure is an error rather than a log line and the
startup line reports the real port. On a signal it flips readiness off, waits `PRE_STOP_DELAY`, and
calls `http.Server.Shutdown` with `SHUTDOWN_TIMEOUT`; in-flight requests finish, and the `stopped`
log line reports how long the drain took.

```jsonc
{"level":"INFO","msg":"listening","addr":"[::]:8081","version":"dev","commit":"dev","buildDate":"dev","config":"host=0.0.0.0 port=8081 logLevel=info ..."}
{"level":"INFO","msg":"shutting down","preStopDelay":0,"timeout":15000000000}
{"level":"INFO","msg":"stopped","drain":1206358}
```
