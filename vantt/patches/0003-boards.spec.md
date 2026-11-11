# Patch 0003 — boards

feat(boards): lanes, parent chips, status age, child panel

One patch per feature since 2026-11-11 (owner decision). Made from the old patches 0010 parent-child, 0012 status-age, 0013 child-panel, 0014 board-lanes; each part below is that patch's spec, unchanged. Code comments still name the old numbers (`vantt 0019`, `patch 0020`): the table in `series.toml` maps them.

## Tests (whole patch)
`cd server && go run ./cmd/migrate up >/dev/null && go test ./internal/handler/ -run 'StatusSince' -count=1 && cd .. && pnpm install --frozen-lockfile --ignore-scripts >/dev/null && (cd packages/core && npx vitest run board-gates) && (cd packages/views && npx vitest run issues locales)`

## Part: old patch 0010 — parent-child (F7) · design D9

### Requirement
A sub-issue shows its parent on board cards (next to the identifier) and list rows
(after the title): a chip with the parent's identifier and title, tooltip
"Sub-issue of <ID> · <title>". Parents keep upstream's sub-issue progress ring.
Reference: Jira's epic link on cards; Linear's parent label on sub-issues.

### Touch points
`packages/views/issues/components/parent-issue-chip.tsx` (+ test), one import and one
element each in `board-card.tsx` and `list-row.tsx`, `issues.parent_chip.title` in
every locale.

### On conflict
Re-insert the chip next to the identifier on cards and after the title on rows.

### Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run issues/components/parent-issue-chip.test.tsx issues/components/board locales`

## Part: old patch 0012 — status-age (F4, F7) · designs D3, D9

### Requirement
`GET /api/issues/status-since[?project_id=]` → `{issues: {<id>: {status, since}}}` for
every issue of the workspace (or project) whose status is not `done`/`cancelled`:
`since` = the issue's last `status_changed` activity, else its creation. Clients show
"days in column" and compare `status` with the issue's current status (a stale entry
after a move shows nothing). Core `@multica/core/board-gates`: `statusSinceOptions`,
`daysInColumn`, and `gateStatuses` (steps handled by a person that have a next status,
for the board's Awaiting approval column). Reference: Jira's days in column.

### Invariants
Read-only; no migration (uses `activity_log`, indexed by issue). Members only.

### Touch points
`server/internal/handler/vantt_status_since.go` (+ test), one route in
`server/cmd/server/router.go` (under `/api/issues`, before `/{id}`),
`packages/core/board-gates/` (+ test), `packages/core/package.json` export,
`listIssueStatusSince` in `packages/core/api/client.ts`.

### On conflict
Re-add the route next to `/decisions/open`; the client method next to the other vantt
methods.

### Tests
See series.toml.

## Part: old patch 0013 — child-panel (F7) · design D9

Needs 0012 (status-age).

### Requirement
On a parent issue's sub-issue panel: next to the upstream done ring and `x/y`, a bar
of the children by status (one segment per status in the status colour; done
category first; tooltip "2 Done, 1 In review…"). Each open child row shows the days
it has sat in its status from one day on, red from five (tooltip "6 days in
In review"). Reference: Jira's "Child issues" panel.

Upstream already gives the rest of D9: parent in the breadcrumb with its `x/y`, the
done ring, stage groups. Patch 0010 adds the parent chip on cards and rows.

### Touch points
`packages/views/issues/components/child-panel-extras.tsx` (+ test), in
`issue-detail.tsx` one import, `<ChildStatusBar>` after the header's done count and
`<DaysInStatus>` at the end of `SubIssueRow`'s link; `child_panel.*` in every locale.

### On conflict
Keep the bar beside the sub-issue header's progress, and the days at the trailing end
of each child row before due date/assignee.

### Tests
See series.toml.

## Part: old patch 0014 — board-lanes (F4) · designs D3, D3d "Automatic"

Needs 0005–0007 (workflow rules), 0012 (status-age), 0013 (DaysInStatus).

### Requirement
A board grouped by status that shows issues of several workflows — a project whose
Type rules use a workflow other than the project's, or the workspace board ("All
issues", D3) when the workspace has more than one workflow — shows **five lanes**,
each holding several statuses (Backlog = built-in `backlog`, not yet committed, owner idea 2026-11-10):
To do (unstarted), In progress (started), **Awaiting approval** (every step a person
handles that has a next status, from all the workflows; lock icon in the heading),
Done (done and closed). Lanes without a visible status are not shown.
- Lane headings wear Multica's own status icons (backlog, todo, in_progress, in_review for Awaiting approval, done), styled like status columns.
- Cards name their real status (built-in statuses get a chip; custom ones already do)
  and their goal (linked to the issue, else its parent, else its project).
- Awaiting approval lists the longest wait first.
- The workspace board loads the steps of every workflow (`useColumnWorkflow`), so no
  issue is hidden because its status is not in Default.
- Every card shows days in column (from one day; red from five).
- Dropping on a lane: the card keeps its status if it is already in the lane, else
  takes the lane's first status its own workflow allows (the Type rule's workflow,
  else the project's); no such status → refused with a toast.
- "+" in a lane creates in its first status; none in Awaiting approval.
- Automatic (D3d): the lanes show only while the cards on the board (after filters)
  follow several workflows; when they follow one (e.g. Filter → Type = Tech · *), the
  board shows that workflow's steps as columns, as a board of one workflow does (D3b).
Reference: Jira board column mapping (several statuses per column), days in column.

Fixes the deviation in 0007: that patch showed every status of every workflow as its
own column on such boards; D3d asked for status groups.


### Touch points
`packages/views/issues/components/board-lanes.tsx` (+ test); `board-view.tsx`
(lane groups, merged pages, rule-aware `workflowOfIssue`, lane resolution in drag over
and drag end, `BoardLaneContext` provider); `board-column.tsx` (`lane`, `statuses` on
`BoardColumnGroup`, `LaneHeading`); `board-card.tsx` (status chip, days);
`utils/drag-utils.ts` `buildColumns` (several statuses per column, a column's own order);
`utils/workflow-columns.ts` (workspace board: every workflow's steps); `board_lanes.*` in
every locale.

### On conflict
Board grouping code moves often upstream. Keep: lanes built from the visible statuses,
cards bucketed by status → lane, lane resolved to one concrete status before any of
upstream's drop checks run.

### Tests
See series.toml.
