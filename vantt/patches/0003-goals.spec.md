# Patch 0003 — goals (F2, server) · design D1, D2

UI is patch 0004 goals-ui.

## Reference
Atlassian Goals (status On track / At risk / Off track, regular updates, goal tree),
Linear Initiatives (projects roll up to an initiative). Paperclip `goals` (level, parent,
owner) — without numbers; we add target/current/unit/direction/due.

## Requirement
### Data (add-only, no foreign keys, one CONCURRENTLY index per migration)
- `goal`: id, workspace_id, parent_id?, title, description, unit, direction
  (`increase` | `decrease`), start_value, target_value, current_value, due_date?,
  status (`planned` | `active` | `achieved` | `missed` | `cancelled`), measured_by
  (`manual` | `script`), owner_type?/owner_id?, current_updated_at, position,
  created_at, updated_at.
- `goal_value`: id, goal_id, value, note, actor_type, actor_id, created_at — the history.
- `goal_link`: goal_id, target_type (`project` | `issue`), target_id, created_at — the
  work that moves a goal (an epic issue, a whole project).

### Health (computed, never stored)
- `progress` = (current − start) / (target − start), clamped 0..1 (direction-aware).
- `expected` = elapsed share of [created_at, due_date].
- ratio = progress / expected: `on_track` ≥ 0.9, `at_risk` ≥ 0.6, else `off_track`; before 5%
  of the time is used everything is `on_track`. No due date → `no_target_date`.
- `update_due` when measured_by = manual and current_updated_at older than 7 days.
- Closed statuses (achieved, missed, cancelled) report their status instead.

### API (workspace-scoped, members read, owner/admin or goal owner write)
- `GET/POST /api/goals`, `GET/PATCH/DELETE /api/goals/{id}` (delete refused while the goal has
  child goals; its links and history are deleted with it).
- `POST /api/goals/{id}/values` `{value, note}` → appends history, updates current.
  Agents may call it (Phương's daily script) when `measured_by = script`.
- `PUT /api/goals/{id}/links` `{links:[{target_type,target_id}]}`.
- Links come with every goal in the list, so clients resolve chips without another call.

### UI (D1, D2)
- Sidebar "Goals". List: summary tiles (off track, on track, update due, open work without
  a goal), tree company → project goals, each row: id, title, scope, due, current/target
  with bar, health pill + pace text ("needs 6.3/wk, doing 1.5/wk"), measured by + age of
  the number, linked work count.
- Detail: header with health; big number; chart actual vs needed line; measured by;
  blocking work (linked issues waiting on a person); work list; history; Enter number.
- Issue page and board card show the goal chip of a linked issue (or its parent's).

## Invariants
1. No goals = no visible change except the sidebar entry.
2. Add-only schema; dropping the patch leaves unused tables.
3. Strings in every locale.

## Workspace deletion
The tables have no foreign keys, so `pkg/db/queries/workspace_delete.sql` deletes their rows
(`deleted_goal_links`, `deleted_goal_values`, `deleted_goals`, after `deleted_project_resources`)
and `workspace_delete_manifest_test.go` lists them as `workspaceDelete`.

## Touch points (expected)
- `server/migrations/NNN_goal*.sql`, `server/pkg/db/queries/goal.sql` + sqlc output,
  `server/internal/handler/goal.go` (+ tests), router registration.
- `packages/core/goals/*` (api, schemas + malformed tests, queries, mutations),
  `packages/core/paths` (goals route), `packages/views/goals/*`, sidebar nav,
  web route `apps/web/app/[workspaceSlug]/(dashboard)/goals`, desktop `routes.tsx`,
  locales.

## On conflict
Migrations: renumber to after upstream's latest. Router/sidebar/routes: re-add the single
registration lines. Keep the API and table names.
