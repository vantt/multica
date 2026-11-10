# Patch 0010 — parent-child (F7) · design D9

## Requirement
A sub-issue shows its parent on board cards (next to the identifier) and list rows
(after the title): a chip with the parent's identifier and title, tooltip
"Sub-issue of <ID> · <title>". Parents keep upstream's sub-issue progress ring.
Reference: Jira's epic link on cards; Linear's parent label on sub-issues.

## Touch points
`packages/views/issues/components/parent-issue-chip.tsx` (+ test), one import and one
element each in `board-card.tsx` and `list-row.tsx`, `issues.parent_chip.title` in
every locale.

## On conflict
Re-insert the chip next to the identifier on cards and after the title on rows.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run issues/components/parent-issue-chip.test.tsx issues/components/board locales`
