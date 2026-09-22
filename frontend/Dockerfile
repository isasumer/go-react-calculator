# syntax=docker/dockerfile:1.19
#
# Multi-stage build for the calculator SPA (ADR-0009).
#
#   deps    : npm ci from the lockfile, with ~/.npm mounted as a cache so a
#             rebuild does not re-download the registry.
#   build   : tsc -b && vite build → /app/dist
#   runtime : nginxinc/nginx-unprivileged:alpine-slim — nginx master and
#             workers all run as uid 101 and listen on 8080, so nothing in the
#             image ever needs root or CAP_NET_BIND_SERVICE.
#
# The image ships static files plus one config template; the backend address
# is chosen at start time (BACKEND_UPSTREAM), so the same artefact runs in
# compose and anywhere else.

# ------------------------------------------------------------------- deps ---
FROM node:22-alpine AS deps

WORKDIR /app

# Only the lockfile pair, so this layer survives every source change.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

# ------------------------------------------------------------------ build ---
FROM node:22-alpine AS build

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Baked into the bundle by Vite. VITE_API_BASE_URL is deliberately left unset:
# frontend/src/config.ts already defaults to "/api", which is the same-origin
# path nginx proxies below.
ARG VITE_APP_VERSION=dev
ENV VITE_APP_VERSION=${VITE_APP_VERSION}

RUN npm run build

# ---------------------------------------------------------------- runtime ---
FROM nginxinc/nginx-unprivileged:alpine-slim

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="go-react-calculator-frontend" \
      org.opencontainers.image.description="Calculator SPA (React + Vite) served by unprivileged nginx with an /api reverse proxy" \
      org.opencontainers.image.source="https://github.com/isasumer/go-react-calculator" \
      org.opencontainers.image.documentation="https://github.com/isasumer/go-react-calculator/blob/main/frontend/README.md" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.base.name="nginxinc/nginx-unprivileged:alpine-slim"

# Rendered into /etc/nginx/conf.d/default.conf at start by the base image's
# envsubst entrypoint, replacing the stock default.conf of the same name.
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY nginx/security-headers.conf /etc/nginx/security-headers.conf
COPY --from=build /app/dist /usr/share/nginx/html

ENV BACKEND_UPSTREAM=backend:8081

# Inherited from the base, stated here so `docker inspect` shows it.
USER 101

EXPOSE 8080

# busybox wget ships in the alpine base (there is no curl); --spider makes it
# a HEAD-like probe that exits non-zero on any status other than 2xx.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["wget", "--spider", "-q", "http://127.0.0.1:8080/healthz"]
