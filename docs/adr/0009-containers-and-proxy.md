# ADR-0009: Ship distroless non-root images behind a same-origin nginx proxy

- **Status:** Accepted
- **Date:** 2026-09-23
- **Ticket:** #19

## Context
The assignment has to run on a reviewer's machine with one command, and the way it runs there should be
the way it would run in production — otherwise the container work is theatre. Two forces pull on the
design. The first is attack surface: a Go service needs nothing from a distribution at runtime, so any
shell, package manager or libc in the final image is only something to patch. The second is the browser:
the SPA has to reach the API, and the two obvious options are a CORS allowlist (the API is published and
the browser calls it cross-origin) or a reverse proxy (the API is private and the browser only ever talks
to one origin). ADR-0005's problem+json responses and the `X-Request-ID` correlation the backend already
implements both assume something in front that behaves like a real edge.

## Decision
The backend image is a multi-stage build: `golang:1.27-alpine` compiles a static `CGO_ENABLED=0` binary
with `-trimpath` and the same `-X` version stamps `backend/Makefile` uses, and the final stage is
`gcr.io/distroless/static-debian12:nonroot` containing that one file. Because distroless has no shell and
no `curl`, the binary probes itself: `server -healthcheck` GETs `http://127.0.0.1:$PORT/readyz` with a
2 s deadline and exits 0 or 1, and that is what both `HEALTHCHECK` and compose's healthcheck run.

The frontend image builds the Vite bundle with `node:22-alpine` and serves it from
`nginxinc/nginx-unprivileged:alpine-slim`, listening on 8080 as uid 101. That nginx is also the edge: it
terminates the only published port, serves the SPA with a one-year immutable cache for hashed
`/assets/` and `no-cache` for `index.html`, sets the security headers (nosniff, `DENY`, `no-referrer`,
a minimal `Permissions-Policy`, and a CSP with `script-src 'self'` — the Vite output has no inline
script, which is asserted against `dist/index.html`), and reverse-proxies `/api/` to the backend,
forwarding the caller's `X-Request-ID` when it sent one and nginx's `$request_id` otherwise. The upstream
is `${BACKEND_UPSTREAM}` (default `backend:8081`), substituted into the config at container start by the
base image's envsubst entrypoint, so one image runs in compose and anywhere else.

`compose.yaml` publishes the frontend only. The backend has no host port: it is reachable solely over the
private compose network, runs with a read-only root filesystem, `cap_drop: [ALL]`,
`no-new-privileges:true` and memory/CPU limits, and the frontend waits for its healthcheck before
starting. Because the browser therefore talks to exactly one origin, **production needs no CORS at all**
— `CORS_ALLOWED_ORIGINS` stays empty and the allowlist exists only for `make dev`, where Vite and the Go
server are on different ports. TLS is not terminated in either image: these run behind an ingress, a load
balancer or a CDN that owns the certificates, which is also why nginx sets no HSTS header.

## Consequences
- The backend image is ~17 MB uncompressed (~5.5 MB to pull) and contains no shell, so a container
  escape has nothing to execute and `docker run --network none` still serves — verified.
- The healthcheck costs one extra flag in `cmd/server` instead of a second binary or a curl in the image,
  and it is unit-tested against `httptest` because the URL is injectable.
- One published port is the whole ingress surface. `/metrics`, `/healthz` and `/readyz` on the backend are
  not proxied, so they stay internal; a scraper reaches them on the compose network.
- Debugging the API directly needs a deliberate step (`cp compose.override.example.yaml
  compose.override.yaml`), and `scripts/smoke.sh` asserts the port is closed without it.
- A read-only root filesystem means anything the backend later wants to write (a cache, a temp file) must
  become an explicit `tmpfs` mount — a change that should be noticed.
- `style-src` still carries `'unsafe-inline'`: Radix primitives set `style=""` attributes at runtime and
  CSP level 3 counts those as inline styles. Tightening it needs `style-src-attr` support and a frontend
  change; it is a follow-up, not a silent hole.
- nginx resolves `BACKEND_UPSTREAM` once at start, so the frontend container must start after the backend
  is healthy — which `depends_on: condition: service_healthy` already guarantees.

## Alternatives considered
- **Alpine or Debian base for the backend** → a shell and a package manager we would have to keep patched,
  for a static binary that needs neither.
- **`scratch`** → 4 MB smaller, but no CA bundle, no `/etc/passwd` for a non-root uid and no timezone data;
  distroless gives those for a couple of MB.
- **Publish the API and use a CORS allowlist** → a second public port, a preflight on every call, and an
  allowlist to get wrong per environment. The proxy removes the problem instead of configuring it.
- **A separate edge container (Caddy/Traefik) in front of both** → a third image and a second config
  language for a stack whose only route is `/api/`; nginx already serves the static files.
- **`curl`/`wget` in the final image for the healthcheck** → reintroduces the userland distroless removes,
  to do what 40 lines of Go already do.
- **Bake the backend URL into the bundle at build time** → the image stops being environment-independent
  and every deployment needs its own build.
