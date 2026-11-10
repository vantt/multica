---
title: vantt/multica fork — minimal Multica patches that follow upstream
status: done except phase 09 (budgets, deferred by the owner)
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
| [01](phase-01-foundation.md) | — | Fork, upstream remote, catchup, CI, images | done |
| [02](phase-02-fork-identity.md) | F0 | Fork version shown; owner instance runs fork images | done (v0.7.2-vantt.3 live) |
| [03](phase-03-owner-gates.md) | F1 | Server-enforced owner gates, Waiting on me, approval panel | done: 0001 server, 0002 panel, 0015 PRs + Approve on GitHub; Waiting on me = saved view from packages (no patch) |
| [04](phase-04-goals.md) | F2 | Goals with numbers | done: 0003 server, 0004 UI; the project repo is the source (plans/goals.md), two-way, repo wins |
| [05](phase-05-workflow-by-type.md) | F3 | Workflow chosen by a select property (Type) | done: 0005 server, 0006 UI; opc.mjs `workflow_rules` |
| [06](phase-06-boards.md) | F4 | Jira-style boards | done: 0007 columns for every rule workflow, 0014 lanes (To do · In progress · Awaiting approval · Done) for boards of several workflows + days in column; saved boards/swimlanes/filters = Multica built-ins (see 0007 spec) |
| [07](phase-07-decisions.md) | F5 | Structured decisions, one-tap answers | done: 0008 server, 0009 UI |
| [08](phase-08-parent-child-reader.md) | F7, F8 | Parent/child clarity; artifact reading page | done: 0010, 0011; D9 round 2 (owner first, 2026-11-10): 0012 status-age, 0013 child-panel; reader round 2 later |
| [09](phase-09-budgets.md) | F6 | Budgets with caps | **deferred** (owner, 2026-11-10) |
| [10](phase-10-upstream-catchup.md) | — | Review upstream v0.7.2, catch up | done: reviews/v0.7.2.md, pin v0.7.2 |

Each phase: read the design, write the patch spec (`vantt/patches/NNNN-<id>.spec.md`),
implement on a branch from `vantt/<tag>`, test (`make test`, `pnpm typecheck`, `pnpm test`),
export the patch, add it to `series.toml`, let CI build, deploy to the owner instance,
check the "Done when" list, then mark the phase done here.
