# Phase 06 — F4 boards (K8) · design D3, D3b, D3c, D3d

## Reference
Jira boards (saved filter + column mapping, swimlanes, quick filters, days in column),
Jira team-managed boards and Linear (columns = statuses / fixed status categories).

## Requirement
Saved boards per project (tabs): name, filter, swimlane, column mode. Default "All issues".
Columns **Automatic** (default): one workflow → its statuses; several → status categories
plus "Awaiting approval" (all owner-gate statuses); new statuses place themselves.
**Custom**: drag statuses into columns, Unmapped hidden. Swimlanes none/Domain/Goal/
Assignee/Parent, collapsible, expand/collapse all, cells scroll. Quick filters on select
properties and assignee. Card: lock, parent epic, real status chip, Type, Goal, days in column.

## Steps
- [ ] Read current board code in `packages/views` and how statuses/categories are exposed.
- [ ] Spec `0004-boards.spec.md`; table `project_board`.
- [ ] Views + core; keep the current board as "All issues" automatic when nothing is set.
- [ ] fgOPC: `.fgopc.yaml` `boards:`; `opc.mjs` creates boards; lint shared status names.

## Done when
momentry has All issues, Tech team, Marketing team boards from the packages, matching D3–D3d.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
