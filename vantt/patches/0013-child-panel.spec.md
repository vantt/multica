# Patch 0013 — child-panel (F7) · design D9

Needs 0012 (status-age).

## Requirement
On a parent issue's sub-issue panel: next to the upstream done ring and `x/y`, a bar
of the children by status (one segment per status in the status colour; done
category first; tooltip "2 Done, 1 In review…"). Each open child row shows the days
it has sat in its status from one day on, red from five (tooltip "6 days in
In review"). Reference: Jira's "Child issues" panel.

Upstream already gives the rest of D9: parent in the breadcrumb with its `x/y`, the
done ring, stage groups. Patch 0010 adds the parent chip on cards and rows.

## Touch points
`packages/views/issues/components/child-panel-extras.tsx` (+ test), in
`issue-detail.tsx` one import, `<ChildStatusBar>` after the header's done count and
`<DaysInStatus>` at the end of `SubIssueRow`'s link; `child_panel.*` in every locale.

## On conflict
Keep the bar beside the sub-issue header's progress, and the days at the trailing end
of each child row before due date/assignee.

## Tests
See series.toml.
