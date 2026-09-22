# Prompts log

The assignment asks for the prompts used. This file is appended by every working session, verbatim, with
what was accepted, what was rejected and why, and what was written by hand. Tool: Claude Code (Anthropic).
Prompts were written in Turkish; an English gloss follows each one in brackets. The company name is
redacted as `[company]` because this repository is public.

Format for each session:

```
## Session <ticket-id> — <YYYY-MM-DD>
1. <prompt, verbatim>
   [English gloss]
**Accepted** — …
**Rejected (why)** — …
**Written by hand** — …
```

---

## Session PLANNING — 2026-09-22

1. `bana [company] dan böyle bir metin geldi analiz et` + the assignment text pasted verbatim.
   [I received this text from [company], analyse it.]
2. `go yu kur ama ben önce end to end bir epic oluşturmanı istiyorum. yani önce bir tane plan ve sprint oluşturmamız lazım. bir github reposu oluşturmamız lazım. Daha sonra hem fe hem de backend için ticketlar sub ticket lar follw up lar açmamız lazım . yani önce planlama gerekiyor. ayrıca front-end için bununla aynı dosyada olan lived projesi baz alınabilir dosyalama UI, helpers, api callings, state management, local storage etc. backend için production grade solution lazım. yani sadece bu task i implement etmeyeceğiz senior olduğumuzu göstermemiz gerekiyor. Benim aklıma gelmeyen ve veya yapmamız gereken implementasyon planlama vs varsa ilerletelim. Her feature ı section ı veya ticket ı yeni pr da ve yeni session da implement edeceğim. dolayısıyla her session taze scope ile başlamış olacak. Senden detaylı implementasyon planı istiyorum ingilizce olsun`
   [Install Go, but first create an end-to-end epic: a plan and sprints, a GitHub repository, tickets / sub-tickets / follow-ups for both frontend and backend. Base the frontend structure (files, UI, helpers, API calls, state management, local storage) on an existing production project of mine. The backend must be a production-grade solution; we are not just implementing the task, we are demonstrating seniority. Add anything I have not thought of. Every ticket will be implemented in a new PR and a new session with fresh scope. Give me a detailed implementation plan in English.]

**Accepted** — The AI's assessment of what the assignment actually evaluates; the sprint structure with a
"submittable checkpoint" at the end of Sprint 2; the frozen API contract; the decision to use Go despite no
production Go experience; the ADR list; the per-issue session protocol captured in `CLAUDE.md`; the 40
tickets and 6 epics generated from a single manifest so issues and `docs/PLAN.md` cannot drift.

**Rejected (why)** — Nothing was rejected outright. The plan was shaped by the second prompt: the AI's
first analysis proposed a 4-hour build; the direction was changed to production-grade with the extra
investment confined to operational maturity rather than features, and the frontend conventions were
pinned to the author's existing codebase instead of the AI's defaults.

**Written by hand** — The direction and constraints above. The plan text was generated and reviewed
before commit; later sessions refine it only through follow-up issues.

## Session P0-01 — 2026-09-22

1. `issue #1'i implement et`
   [Implement issue #1.]

**Accepted** — Everything in the issue scope as generated: `.editorconfig`, MIT `LICENSE`, `CODEOWNERS`,
PR and issue templates carrying the Definition of Done, root `Makefile` whose targets fail fast naming the
ticket that will provide them, ADR template + ADR-0001 (monorepo) + ADR-0002 (frontend stack), ADR index,
`errors.md` and `ARCHITECTURE.md` stubs, this prompts log, README links, `CLAUDE.md` refinements.

**Rejected (why)** — n/a. The session was checked against the issue's acceptance criteria and the
out-of-scope list (no application code, no CI workflows).

**Written by hand** — None in this session; review happens on the PR before merge.
