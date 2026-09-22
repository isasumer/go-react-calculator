# syntax=docker/dockerfile:1.19
#
# Multi-stage build for the calculator API (ADR-0009).
#
#   builder : golang:1.27-alpine, static CGO-free binary, caches mounted so a
#             rebuild after a source-only change does not re-download or
#             re-compile the module graph.
#   final   : gcr.io/distroless/static-debian12:nonroot — no shell, no package
#             manager, no libc to keep patched, runs as uid 65532. The image
#             holds exactly one file we put there: /server.
#
# Build:
#   docker build -t calc-backend \
#     --build-arg VERSION="$(git describe --tags --always --dirty)" \
#     --build-arg COMMIT="$(git rev-parse --short HEAD)" \
#     --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" backend

# ---------------------------------------------------------------- builder ---
FROM golang:1.27-alpine AS builder

WORKDIR /src

# CGO off makes the binary static, so the distroless *static* base is enough.
# -trimpath strips local paths out of the binary, which is what makes two
# builds of the same commit byte-identical. GOTOOLCHAIN=local refuses to
# silently download another Go version mid-build.
ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath \
    GOTOOLCHAIN=local

# Dependencies first: this layer is reused until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Stamped into internal/observability, exactly as backend/Makefile does it —
# `server -version`, the /version endpoint and the build_info metric all read
# these. Defaults keep a plain `docker build` working.
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ENV BUILDINFO=github.com/isasumer/go-react-calculator/backend/internal/observability

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
      -ldflags "-s -w -X ${BUILDINFO}.Version=${VERSION} -X ${BUILDINFO}.Commit=${COMMIT} -X ${BUILDINFO}.BuildDate=${BUILD_DATE}" \
      -o /out/server ./cmd/server

# ------------------------------------------------------------------ final ---
FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="go-react-calculator-backend" \
      org.opencontainers.image.description="Calculator REST API (Go, stdlib net/http)" \
      org.opencontainers.image.source="https://github.com/isasumer/go-react-calculator" \
      org.opencontainers.image.documentation="https://github.com/isasumer/go-react-calculator/blob/main/backend/README.md" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.base.name="gcr.io/distroless/static-debian12:nonroot"

COPY --from=builder /out/server /server

# uid/gid 65532 in the distroless base. Named here as well so `docker inspect`
# reports it and nothing can start this image as root by accident.
USER nonroot:nonroot

EXPOSE 8081

# The base has no shell and no curl, so the binary probes itself: it GETs
# http://127.0.0.1:$PORT/readyz and exits 0 only when the server reports ready.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "-healthcheck"]

ENTRYPOINT ["/server"]
