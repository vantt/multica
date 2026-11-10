# Patch 0012 — status-age (F4, F7) · designs D3, D9

## Requirement
`GET /api/issues/status-since[?project_id=]` → `{issues: {<id>: {status, since}}}` for
every issue of the workspace (or project) whose status is not `done`/`cancelled`:
`since` = the issue's last `status_changed` activity, else its creation. Clients show
"days in column" and compare `status` with the issue's current status (a stale entry
after a move shows nothing). Core `@multica/core/board-gates`: `statusSinceOptions`,
`daysInColumn`, and `gateStatuses` (steps handled by a person that have a next status,
for the board's Awaiting approval column). Reference: Jira's days in column.

## Invariants
Read-only; no migration (uses `activity_log`, indexed by issue). Members only.

## Touch points
`server/internal/handler/vantt_status_since.go` (+ test), one route in
`server/cmd/server/router.go` (under `/api/issues`, before `/{id}`),
`packages/core/board-gates/` (+ test), `packages/core/package.json` export,
`listIssueStatusSince` in `packages/core/api/client.ts`.

## On conflict
Re-add the route next to `/decisions/open`; the client method next to the other vantt
methods.

## Tests
See series.toml.
