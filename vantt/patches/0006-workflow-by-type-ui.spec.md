# Patch 0006 — workflow-by-type-ui (F3, UI) · design D6

Needs 0005.

## Requirement
Project sidebar row "By Type" under "Workflow": a summary ("Copy → content, …" or "Same
for every Type"); owners/admins open a popover to choose the select property and, per
option, the workflow (or "Project workflow"); Save replaces the rules.
Core: `@multica/core/workflow-rules` (query + mutation), client methods
`listProjectWorkflowRules`, `setProjectWorkflowRules`. Strings `issues.workflows.rules.*`
in every locale.

Deviation from D6: a sidebar popover instead of a settings page, because Multica keeps
project settings in the project's sidebar (Workflow row).

## On conflict
Mount next to `ProjectWorkflowPropRow` in `project-detail.tsx`; re-add the client block
before the goals block.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && (cd packages/views && npx vitest run locales projects workflows)`
