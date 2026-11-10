# Patch 0002 — approval-panel (F1, UI) · design D5b

## Requirement
On the issue page, while the issue sits in a workflow step whose handler is the
viewer (handler `member` with the viewer's id, `project_lead` resolving to the
viewer, or `creator` when the creator is the viewer), show an approval panel above
the title:
- title "Waiting for your approval", the current status, and a note that agents
  cannot move the issue out of this step (patch 0001);
- **Approve** → status = the step's `next_status_key` (or the next column, as
  `workflowNextStatus` computes);
- **Send back** → asks "What needs to change?", posts the text as a comment, then
  status = the step's `back_status_key`. Hidden when the step has no back status.

Reference: Jira approvals (Approve / Decline panel).

## Invariants
1. Nobody but the step's person sees the panel; agent/squad/no-handler steps never
   show it.
2. Approve and Send back go through the page's normal `updateField`, so every
   existing confirm/handoff behaviour applies.
3. Strings exist in every locale (`parity.test.ts`).
4. No server change.

## Touch points
- `packages/views/workflows/approval-panel.tsx` (+ `.test.tsx`): new.
- `packages/views/issues/components/issue-detail.tsx`: import, and
  `<ApprovalPanel …/>` right before `<div id={descriptionSectionId}>`, after
  `IssueDuplicateBanner`.
- `packages/views/locales/*/issues.json`: `workflows.approval.*`.

## On conflict
If the issue page layout moves, mount the panel at the top of the main reading
column, after any banners and before the title. Re-add the locale keys under
`workflows` (or wherever workflow strings move to) for every locale.

## Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run workflows/approval-panel.test.tsx locales`
