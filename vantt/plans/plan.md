---
title: vantt/multica fork — minimal Multica patches that follow upstream
status: in progress
source: fgOPC multica/plans/2026-11-10-multica-fork.md (owner-approved 2026-11-10)
design: ../design/ (approved D1–D7)
---

# Plan

Fork Multica with the fewest possible patches, keep up with upstream through
`vantt/scripts/catchup.mjs`, run it on the owner's own instance.

Rules: no configuration = upstream behaviour; migrations only add; Multica logo, name and
copyright untouched; follow established practice (Jira, Linear, Atlassian Goals) before
inventing; every patch has a spec and aims upstream (proposals upstream need the owner's OK).

| Phase | Patch | What | Status |
|---|---|---|---|
| [01](phase-01-foundation.md) | — | Fork, upstream remote, catchup, CI, images | in progress |
| [02](phase-02-fork-identity.md) | F0 | Fork version shown; owner instance runs fork images | todo |
| [03](phase-03-owner-gates.md) | F1 | Server-enforced owner gates, Waiting on me, approval panel | todo |
| [04](phase-04-goals.md) | F2 | Goals with numbers | todo |
| [05](phase-05-workflow-by-type.md) | F3 | Workflow chosen by a select property (Type) | todo |
| [06](phase-06-boards.md) | F4 | Jira-style boards: saved boards, automatic columns, swimlanes, quick filters | todo |
| [07](phase-07-decisions.md) | F5 | Structured decisions, one-tap answers | todo |
| [08](phase-08-parent-child-reader.md) | F7, F8 | Parent/child clarity; artifact reading page (design first) | todo |
| [09](phase-09-budgets.md) | F6 | Budgets with caps | **deferred** (owner, 2026-11-10) |
| [10](phase-10-upstream-catchup.md) | — | Review upstream v0.7.2, catch up, keep patches green | todo |

Each phase: read the design, write the patch spec (`vantt/patches/NNNN-<id>.spec.md`),
implement on a branch from `vantt/<tag>`, test (`make test`, `pnpm typecheck`, `pnpm test`),
export the patch, add it to `series.toml`, let CI build, deploy to the owner instance,
check the "Done when" list, then mark the phase done here.
