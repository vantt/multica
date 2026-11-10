# Phase 09 — F6 budgets — DEFERRED (owner, 2026-11-10; not scheduled)

## Reference
Paperclip `budget_policies`, `budget_incidents`.

## Requirement
Monthly cost caps per agent and per project from Multica's usage data. Over the cap, new
tasks are not claimed (reason given) and a decision is opened for the owner (raise or stop).

## Steps
- [ ] Spec `0009-budgets.spec.md`; design D8 sketch first, owner approves.

## Done when
A capped agent stops taking work and the owner gets one decision.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
