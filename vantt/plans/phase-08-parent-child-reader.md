# Phase 08 — F7 parent-child, F8 artifact-reader · design D9, D10 (to draw)

## Requirement
D9: in list, board and issue page you can tell an epic/parent from a child and see which
parent a child belongs to (find the reference pattern first: Jira epics/child issues,
Linear sub-issues). D10: reading page for documents (spec, copy, reports) with good long
reading: width, type scale, table of contents, approved-version mark (references: GitHub
markdown view, Notion/Confluence page, Paperclip documents).

## Steps
- [ ] Draw D9, D10 on the design canvas from the reference patterns; owner approves.
- [ ] Specs `0007-parent-child.spec.md`, `0008-artifact-reader.spec.md`; UI only.

## Done when
Owner approves the designs and both run on the owner instance.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
