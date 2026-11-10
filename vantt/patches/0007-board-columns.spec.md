# Patch 0007 — board-columns (F4) · design D3, D3b, D3c, D3d

Needs 0005/0006 (workflow rules).

## Requirement
A project board's columns, when the project has workflow rules (by Type), are the
project workflow's steps plus each rule workflow's middle steps it lacks, inserted
before the closing steps (blocked, done, cancelled). This is design D3d "Automatic"
for a board with several workflows. One workflow: unchanged.

## What the rest of D3–D3d uses instead (no patch, Multica already has it)
Following "best practice first / reuse before inventing":
- Saved boards per team (D3 tabs, D3b "Tech team") = Multica saved views scoped to the
  project (board mode, filters by Domain/Type), pinned to the sidebar; created from
  packages by `opc.mjs` (`views:`).
- Swimlanes (D3c) = Multica's swimlane mode, grouping by assignee/project/property.
- Quick filters = Multica's filter chips (property filters for Type, Domain, Goal).

## Deviations still open (need the owner)
- "Awaiting approval" lifecycle column and days-in-column badge (D3) are not built.
- Custom column mapping (D3d Custom) is not built.

## Touch points
`packages/core/workflow-rules/index.ts` (`mergeRuleWorkflows`, plain options object),
`packages/views/issues/utils/workflow-columns.ts` (`useColumnWorkflow`), its test mocks
`useQuery`.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && (cd packages/core && npx vitest run workflow-rules) && (cd packages/views && npx vitest run issues/utils workflows)`
