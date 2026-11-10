# Patch 0014 — board-lanes (F4) · designs D3, D3d "Automatic"

Needs 0005–0007 (workflow rules), 0012 (status-age), 0013 (DaysInStatus).

## Requirement
A board grouped by status that shows issues of several workflows — a project whose
Type rules use a workflow other than the project's, or the workspace board ("All
issues", D3) when the workspace has more than one workflow — shows **five lanes**,
each holding several statuses (Backlog = built-in `backlog`, not yet committed, owner idea 2026-11-10):
To do (unstarted), In progress (started), **Awaiting approval** (every step a person
handles that has a next status, from all the workflows; lock icon in the heading),
Done (done and closed). Lanes without a visible status are not shown.
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


## Touch points
`packages/views/issues/components/board-lanes.tsx` (+ test); `board-view.tsx`
(lane groups, merged pages, rule-aware `workflowOfIssue`, lane resolution in drag over
and drag end, `BoardLaneContext` provider); `board-column.tsx` (`lane`, `statuses` on
`BoardColumnGroup`, `LaneHeading`); `board-card.tsx` (status chip, days);
`utils/drag-utils.ts` `buildColumns` (several statuses per column, a column's own order);
`utils/workflow-columns.ts` (workspace board: every workflow's steps); `board_lanes.*` in
every locale.

## On conflict
Board grouping code moves often upstream. Keep: lanes built from the visible statuses,
cards bucketed by status → lane, lane resolved to one concrete status before any of
upstream's drop checks run.

## Tests
See series.toml.
