# Security Policy

## Supported versions

Only the `main` branch and the latest tagged release receive security fixes. There is no long-term
support branch; older tags are not patched.

| Version          | Supported |
| ----------------- | --------- |
| `main`             | Yes       |
| latest tag         | Yes       |
| anything older     | No        |

## Reporting a vulnerability

Please report suspected vulnerabilities privately using GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability):
open a draft security advisory under this repository's **Security** tab ("Report a vulnerability").
Do not open a public issue for a suspected vulnerability.

We aim to acknowledge reports within a few days. There is no bug bounty; this is a demonstration
project, not a production service handling real user data.

## Scope

In scope:
- The Go backend (`backend/`) and its HTTP API.
- The React frontend (`frontend/`) and the nginx configuration that serves it.
- The container images and Compose stack used to run both (`compose.yaml`, `backend/Dockerfile`,
  `frontend/Dockerfile`).
- CI/CD workflows under `.github/workflows/`.

Out of scope:
- Denial of service against the demo deployment itself (no SLA is offered).
- Findings that require an already-compromised host or CI runner.
- Third-party dependencies' own disclosed CVEs — report those upstream; here we only track whether
  they are patched (see Dependabot / `npm audit` / `govulncheck` / Trivy below).

## Existing abuse controls

The API is intentionally a stateless pure function with no accounts, sessions, or stored user data,
which removes most of the traditional attack surface (no auth to bypass, nothing to exfiltrate). The
service still applies these controls at the application layer:

- **Rate limiting** — per-client token-bucket limiting (`golang.org/x/time/rate`) on the backend, keyed
  off the client address nginx forwards.
- **Request body limits** — the backend rejects oversized request bodies before decoding.
- **Timeouts** — request-scoped timeouts on every handler, so a slow or stuck client cannot hold a
  goroutine indefinitely.
- **No authentication by design** — there is nothing to authenticate: the API takes an expression and
  returns a result, with no per-user state. This is a deliberate scope decision, not an oversight (see
  `docs/ARCHITECTURE.md` "Threat model").

## Supply-chain scanning

This repository also runs, on every pull request (see `.github/workflows/security.yml`):
- **CodeQL** static analysis for Go and JavaScript/TypeScript.
- **Trivy** image scans of both built containers, blocking on CRITICAL (fixed) findings.
- **gitleaks** secret scanning.
- **npm audit** against frontend dependencies.
- **govulncheck** against the Go module (in `backend.yml`).
- **Dependabot** for weekly dependency updates (Go modules, npm, GitHub Actions, Docker base images).
