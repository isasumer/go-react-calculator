# Backend

Go service for the calculator API. Module: `github.com/isasumer/go-react-calculator/backend`.
It currently has only a placeholder server that returns 404 for every request. Routes arrive in B1-02.

## Layout

```
cmd/server/              composition root; main() calls run(ctx, args, getenv, stdout)
internal/calc/           pure domain (B1-01)
internal/httpapi/        HTTP transport, problem+json (B1-02)
internal/middleware/     middleware chain (B1-04)
internal/config/         env → typed Config (B1-03)
internal/observability/  slog, metrics, build info (B1-03, B1-05)
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
| `make build` | static binary in `bin/server` with version, commit and date set via `-ldflags` |
| `make run` | runs the server on `:8081` (`go run ./cmd/server -addr :9000` to change the address) |

The server logs JSON to stdout, starting with a `listening` line that includes the bound address. It
shuts down gracefully on SIGINT/SIGTERM and exits 0.
