# Patch 0004 — goals-ui (F2, UI) · design D1, D2

Needs patch 0003 goals (API).

## Requirement
- Sidebar "Goals" after Projects; command palette finds it.
- List (D1): tiles Off track / At risk / On track / Update due; goal tree (parents
  first) with title, due date, current/target with bar, health pill (Update due wins),
  pace "needs X/wk, doing Y/wk", measured by, linked work count; New goal form (result,
  start, target, unit, due date, better when higher/lower, parent).
- Detail (D2): health, big number of target, pace; SVG chart of history against the
  dashed needed line; Enter a value with a note; linked issues; history.
Reference: Atlassian Goals, Linear initiatives.

## Invariants
1. Every route is a parameterless WorkspacePaths builder (`goals`) plus `goalDetail(id)`.
2. Strings in every locale (parity test); goal page strings live in `projects.json`
   under `goals`, the nav label in `layout.json` `nav.goals`.
3. API responses go through zod with fallbacks (`core/goals/schemas.ts`).

## Touch points
- New: `packages/core/goals/*`, `packages/views/goals/*`,
  `apps/web/app/[workspaceSlug]/(dashboard)/goals/**`,
  `apps/desktop/src/renderer/src/pages/goal-detail-page.tsx`.
- One-line registrations: `core/api/client.ts` (imports + goal methods block before
  `listProjectResources`), `core/package.json` and `views/package.json` exports,
  `core/paths/paths.ts`, `core/paths/route-icons.ts` (`Target`, `goals` keys and page),
  `core/paths/tab-subject.ts`, `core/paths/consistency.test.ts`,
  `views/layout/app-sidebar.tsx`, `views/layout/route-icon-components.tsx`,
  `views/search/use-root-model.ts` (palette keywords; was `search-command.tsx` before v0.7.2), desktop `routes.tsx`, path mocks in
  3 layout tests, locales.

## On conflict
Re-add each registration next to `projects`. If upstream adds a page registry or
renames NavKey/WorkspacePageKey, register `goals` the same way as `projects`.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && (cd packages/core && npx vitest run goals paths) && (cd packages/views && npx vitest run locales layout search)`
