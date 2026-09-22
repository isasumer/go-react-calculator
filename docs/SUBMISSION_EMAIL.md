# Submission email draft

Draft only — not sent from here. [Recruiter name] and [Company] are placeholders; fill in and send manually.

---

**Subject:** Take-home submission — full-stack calculator (Go + React)

Hi [Recruiter name],

Here's my submission for the [Company] take-home: **https://github.com/isasumer/go-react-calculator**
(public, v1.0.0).

It's a calculator built as a small production system rather than a script: a Go REST microservice behind
a typed JSON contract (OpenAPI 3.1, `problem+json` errors, request IDs, rate limiting, structured logs,
Prometheus metrics) and a React/TypeScript SPA, both containerised behind an unprivileged nginx reverse
proxy, with CI enforcing formatting, linting, type-checking, tests and coverage gates on every PR.

**Run it in one command:**

```sh
docker compose up --build --wait
```

then open <http://localhost:8080>. `make smoke` runs 9 black-box checks against the running stack.

**Where things live:**
- Tests & coverage: `README.md` → Testing (98.1% backend, 100%/98.6%/100%/100% frontend); raw numbers
  reproduce with `make check`.
- AI prompts: every session's prompts are in [`docs/PROMPTS.md`](https://github.com/isasumer/go-react-calculator/blob/main/docs/PROMPTS.md),
  one section per merged PR, with what was accepted, rejected, and written by hand.
- Architecture decisions: [`docs/adr/`](https://github.com/isasumer/go-react-calculator/tree/main/docs/adr) — 9 ADRs, all Accepted.

**Honest time note:** I used AI tooling (Claude Code) throughout, one session per ticket, each producing
one reviewed PR gated by CI — that's what let the scope go beyond a 2–4 h afternoon project. The full
breakdown, including active-time vs. elapsed-time and where the AI was wrong and caught, is in the
[README Time log](https://github.com/isasumer/go-react-calculator/blob/main/README.md#time-log).

Happy to walk through any part of it live — architecture, a specific PR, or the AI workflow itself.

Best,
Isa

P.S. The repository is public and contains no reference to [Company] anywhere in its history.
