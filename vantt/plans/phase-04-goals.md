# Phase 04 — F2 goals (K6) · design D1, D2

## Reference
Atlassian Goals (status On track / At risk / Off track, regular updates), Linear
Initiatives; Paperclip `goals` (level, parent, owner) plus numbers it lacks.

## Requirement
Goal: title, level (workspace, project), parent, owner, **target, current, unit,
direction (higher/lower is better), due date**, status (planned, active, achieved, missed,
cancelled), measured by (manual | script). Health computed from pace vs due date; "Update
due" when a manual number is older than 7 days. Value history kept. Projects and issues
point to a goal. API to set the current value (Phương's script). UI: Goals page (tree,
progress, health, pace text, measured by, work) and goal detail (chart vs needed line,
blocking work, history). Board quick filter by goal.

## Steps
- [ ] Spec `0002-goals.spec.md`; tables `goal`, `goal_value` (add-only, no FKs).
- [ ] Server API + tests; core schema; views (Goals list, detail); sidebar entry.
- [ ] fgOPC: Phương's goal sync moves from the `Goal` property to this API (fgOPC plan G1/G2).

## Done when
Owner sees momentry goals with real numbers and health; Phương updates a script goal daily.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
