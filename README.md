# go-react-calculator

[![backend](https://github.com/isasumer/go-react-calculator/actions/workflows/backend.yml/badge.svg?branch=main)](https://github.com/isasumer/go-react-calculator/actions/workflows/backend.yml)
[![frontend](https://github.com/isasumer/go-react-calculator/actions/workflows/frontend.yml/badge.svg?branch=main)](https://github.com/isasumer/go-react-calculator/actions/workflows/frontend.yml)

Full-stack calculator: a Go REST microservice and a React + TypeScript frontend, built to production standards (typed config, graceful shutdown, structured logs, metrics, rate limiting, RFC 9457 errors, OpenAPI contract, CI gates, hardened containers, e2e tests).

> Work in progress. The implementation plan, sprint breakdown and every ticket live in [`docs/PLAN.md`](docs/PLAN.md); progress is tracked in the GitHub issues and milestones of this repository.

## Overview

_Filled in D4-01._ Plan and sprint breakdown: [`docs/PLAN.md`](docs/PLAN.md). Architecture: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). AI session protocol: [`CLAUDE.md`](CLAUDE.md).

## Quick start

_Filled in D4-01._ Until then: `make help` lists the umbrella targets.

## API

_Filled in D4-01._ Contract: [`docs/PLAN.md` §1.4](docs/PLAN.md#14-api-contract-frozen-after-b1-02-changes-require-an-adr). Error catalogue: [`docs/errors.md`](docs/errors.md).

## Design decisions

_Filled in D4-01._ Decisions are recorded as ADRs: [`docs/adr/`](docs/adr/README.md) — [ADR-0001 monorepo layout](docs/adr/0001-monorepo-layout.md), [ADR-0002 frontend stack](docs/adr/0002-frontend-stack.md).

## Testing

_Filled in B1-07 / F2-07 / D4-01._

## Project structure

_Filled in D4-01._ See [`docs/PLAN.md` §1.1](docs/PLAN.md#11-repository-layout-monorepo-two-independently-buildable-apps).

## Configuration

The backend is configured entirely by environment variables (host/port, log level and format, CORS allowlist, HTTP timeouts, shutdown behaviour, request limits); every one of them, with its default and meaning, is tabulated in [`backend/README.md` § Configuration](backend/README.md#configuration). Frontend build-time variables are filled in D4-01.

## Time log

_Filled in D4-01._

## Prompts

All prompts used, per session, with what was accepted and rejected: [`docs/PROMPTS.md`](docs/PROMPTS.md).

## License

[MIT](LICENSE)
