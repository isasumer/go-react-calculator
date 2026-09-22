Closes #

## Summary
<!-- What changed and why, in three sentences or fewer. Link the ADR if a decision was made. -->

## Test evidence
<!-- Paste the command(s) you ran and the relevant output excerpt (coverage line, test summary). -->
```
make check
```

## Screenshots (UI changes only)

## Prompts used
<!-- Also appended to docs/PROMPTS.md under "## Session <ticket-id> — <date>". -->

## Definition of Done
- [ ] Every acceptance criterion in the issue is ticked
- [ ] `make check` passes locally (lint, vet, type-check, tests, coverage thresholds)
- [ ] New behaviour has tests; coverage gate not lowered (backend 85 %, frontend 85 % lines / 80 % branches)
- [ ] Docs updated where the issue says so (README section, ADR, OpenAPI, errors catalogue)
- [ ] `docs/PROMPTS.md` appended with this session's prompts and what was accepted/rejected
- [ ] No scope creep: out-of-scope findings filed as `follow-up` issues and linked here
- [ ] Conventional Commit title; squash-merge; branch deleted after merge
