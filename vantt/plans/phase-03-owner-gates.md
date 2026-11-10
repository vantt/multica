# Phase 03 — F1 owner-gates (K5) · design D5, D5a, D5b

## Reference
Jira workflow conditions (only some users may perform a transition); Jira Service
Management approvals ("Waiting for my approval", approval panel Approve/Decline);
Paperclip refuses a review move without an approver (`doc/execution-semantics.md`).

## Requirement
1. A status whose workflow step has `handler.role = owner` is a **gate**. An agent actor
   cannot move an issue out of a gate, nor into `done` when the workflow says
   `done_requires: human`. The server answers 403 with a clear message and records the
   attempt in the timeline. People move it as before.
2. Workflow `rules` (`done_requires`, `max_send_backs`) are stored on the workflow, not
   only copied into step instructions.
3. `GET /api/me/waiting`: issues in a gate whose handler is the caller, all projects,
   oldest in column first, with gate kind (spec, PR, decision, copy), project, goal.
4. UI: sidebar "Waiting on me" + list page (D5a); approval panel on issue page (D5b) with
   evidence (CI, review, QA, PR) and Approve → `next`, Send back (+reason comment) → `back`;
   lock on gated status (D5). PR gate Approve also approves the GitHub PR when the owner's
   GitHub is connected, else "Approve on GitHub" opens the PR.

## Done so far
- 0001 owner-gates (server): GateError/PersonGate, 403 `owner_gate` on UpdateIssue, batch skips;
  4 tests; full handler suite green. Upstream already refuses a run moving an issue from a step it
  is not on (`workflowStepMoved`); 0001 covers agents with no run on the issue and project moves.
- 0002 approval-panel (UI): Approve → next status, Send back → comment + back status; all locales.
- Deviation from D5b: Approve does not approve the GitHub PR (Multica has no per-user GitHub
  token); the owner approves on GitHub as before. Evidence list (CI/review/QA) not shown yet.

## Steps
- [ ] Read: `server/internal/issueworkflow`, status-change handler(s), timeline/activity,
      how actor type is known (agent task token vs member session).
- [ ] Read Paperclip's enforcement code before writing the spec.
- [ ] Spec `0001-owner-gates.spec.md`; tests first (Go handler tests with `testutil`).
- [ ] Server, then core (zod schema + malformed-response test), then views.
- [ ] Probe script: agent token tries to leave a gate → 403; owner → ok.
- [ ] Update fgOPC packages if step instructions should mention the gate.

## Done when
On the owner instance an agent cannot move STU/MOM issues out of spec_approval or
in_review; owner approves from the panel; Waiting on me matches the morning brief list.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
