# Backend

Go service for the calculator API. Module: `github.com/isasumer/go-react-calculator/backend`.
It serves `POST /api/v1/calculate` and `GET /api/v1/operations` (contract: [`docs/PLAN.md` §1.4](../docs/PLAN.md#14-api-contract-frozen-after-b1-02-changes-require-an-adr), errors: [`docs/errors.md`](../docs/errors.md)) plus the operational probes below. The middleware chain arrives in B1-04 and metrics in B1-05.

## Layout

```
cmd/server/              composition root; main() calls run(ctx, args, getenv, stdout)
internal/calc/           pure domain (B1-01)
internal/httpapi/        HTTP transport, problem+json (B1-02)
internal/middleware/     middleware chain (B1-04)
internal/config/         env → typed, validated Config with defaults
internal/observability/  build information; metrics (B1-05)
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
| `CORS_ALLOWED_ORIGINS` | *(empty)* | Comma-separated allowlist for the CORS middleware (B1-04); spaces around items are trimmed and empty items dropped. Empty means no cross-origin browser access — production serves the frontend same-origin (ADR-0009). |
| `READ_HEADER_TIMEOUT` | `5s` | `http.Server.ReadHeaderTimeout`: how long a client may take to send the request headers. |
| `READ_TIMEOUT` | `10s` | `http.Server.ReadTimeout`: headers plus body. |
| `WRITE_TIMEOUT` | `10s` | `http.Server.WriteTimeout`: how long a response may take to write. |
| `IDLE_TIMEOUT` | `60s` | `http.Server.IdleTimeout`: how long a keep-alive connection may sit idle. |
| `SHUTDOWN_TIMEOUT` | `15s` | Budget for draining in-flight requests after the pre-stop delay. Must be greater than zero. |
| `PRE_STOP_DELAY` | `0s` | Pause between flipping `/readyz` to 503 and closing the listener. Keep it `0` locally; in Kubernetes set it to about twice the readiness probe period (e.g. `5s`) so the endpoint is removed from every load balancer *before* the process stops accepting connections. |
| `REQUEST_TIMEOUT` | `5s` | Per-request timeout for the timeout middleware (B1-04). |
| `RATE_LIMIT_RPS` | `20` | Sustained requests per second per client for the rate limiter (B1-04). Minimum `1`. |
| `RATE_LIMIT_BURST` | `40` | Burst size for the same limiter. Minimum `1`. |
| `MAX_BODY_BYTES` | `4096` | Maximum accepted request body; larger bodies get `413 PAYLOAD_TOO_LARGE`. |
| `TRUST_PROXY_HEADERS` | `false` | Whether `X-Forwarded-For` / `X-Request-ID` from upstream may be trusted (B1-04). Only enable behind a proxy you control. |

The timeouts accept any Go duration (`750ms`, `5s`, `2m`); `0` disables the `http.Server` ones.
`REQUEST_TIMEOUT`, `RATE_LIMIT_*`, `MAX_BODY_BYTES` and `TRUST_PROXY_HEADERS` are parsed and logged
now but only take effect when B1-04 wires the middleware chain.

## Operational endpoints

These sit outside `/api/v1`: they are for the platform, not for API clients. All three answer
`GET` (and `HEAD`), send `Cache-Control: no-store`, and return a `405` problem with an `Allow`
header for any other method.

| Endpoint | Answers |
|---|---|
| `GET /healthz` | Liveness: `200 {"status":"ok"}` for as long as the process serves, including while it drains — a draining process must not be restarted. |
| `GET /readyz` | Readiness: `200 {"status":"ready"}` while the server accepts work; `503` problem+json with `code: NOT_READY` from the moment shutdown starts, before the listener closes. |
| `GET /version` | `200 {"version","commit","buildDate","goVersion"}` for the running binary. |

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
