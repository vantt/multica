# Patch 0001 — workflow

feat(workflow): owner gates, workflows by Type and their board columns, approval panel, send-back limit

One patch per feature since 2026-11-11 (owner decision). Made from the old patches 0001 owner-gates, 0002 approval-panel, 0005 workflow-by-type, 0006 workflow-by-type-ui, 0007 board-columns, 0015 approval-pr, 0016 type-first, 0017 derived-property, 0018 send-back-limit; each part below is that patch's spec, unchanged. Code comments still name the old numbers (`vantt 0019`, `patch 0020`): the table in `series.toml` maps them.

## Shared test helpers
`goalRequest` (a request in the test workspace) and `runningTask` live in
`vantt_owner_gates_test.go`. The workflow-rules and send-back tests here and the goals
and decisions tests in later patches use them, so this patch stays first.

## Tests (whole patch)
`cd server && go run ./cmd/migrate up >/dev/null && go test ./internal/handler/ -run 'OwnerGate|AgentStepIsNotAGate|WorkflowRule|FollowsRuleWorkflow|ChangingType|Derived|PropertyDefinition|SendBack|WorkspaceDeletion' -count=1 && cd .. && pnpm install --frozen-lockfile --ignore-scripts >/dev/null && (cd packages/core && npx vitest run workflow-rules) && (cd packages/views && npx vitest run workflows projects modals issues/utils issues/components/issue-detail locales)`

## Part: old patch 0001 — owner-gates (F1, server)

### Requirement
A workflow step whose handler resolves to a **member** (a person: `member`, or
`project_lead`/`creator` when that resolves to a member) is an **owner gate**.
An **agent** actor (request carries a task token: `X-Actor-Source: task_token`)
may not move an issue that sits in a gate **out** of it:
- changing its status → `403 {"code":"owner_gate","status":"<current>","error":"…"}`;
- moving it to another project → same 403 (a project move remaps the status);
- the batch endpoint skips such an issue for that agent and applies the others.

References: Jira workflow conditions (only some users may perform a transition);
Paperclip refuses a review move without an approver.

### Invariants
1. Members are never affected.
2. Agents still move issues **into** a gate (that is how work reaches the owner) and
   change any other field of a gated issue (title, description, labels…).
3. Steps handled by an agent, a squad, or with no handler are not gates.
4. Without member-handled steps the behaviour equals upstream.
5. The check runs before `applyIssueWorkflow`, using the issue's **current**
   project workflow and status (`prev`), not the requested ones.

### Touch points
- `server/internal/issueworkflow/issueworkflow.go`: `GateError`, `PersonGate`
  (placed before `MoveTarget`).
- `server/internal/handler/issue_workflow_entry.go`: `writeIssueWorkflowError`
  maps `GateError` → 403; `refuseAgentLeavingGate` (before `agentRunOnIssue`).
- `server/internal/handler/issue.go`: `UpdateIssue` and `BatchUpdateIssues` call
  `refuseAgentLeavingGate` right after `callerType` is resolved, before
  `applyIssueWorkflow`.
- `server/internal/handler/vantt_owner_gates_test.go`: new tests.

### On conflict
Upstream moves or renames the workflow write path: put the gate check wherever
the caller type is resolved and before the workflow is applied, for every write
path that can change `status` or `project_id` (single update, batch update, and
any new one). Keep the 403 code `owner_gate`. If upstream adds its own
role/permission gating for workflow steps, compare and prefer theirs; mark this
patch `merged`/drop it when it covers invariants 1–4.

### Tests
`cd server && go test ./internal/handler/ -run 'OwnerGate|AgentStepIsNotAGate' -count=1`
(needs `DATABASE_URL` to a migrated test database).

## Part: old patch 0002 — approval-panel (F1, UI) · design D5b

### Requirement
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

### Invariants
1. Nobody but the step's person sees the panel; agent/squad/no-handler steps never
   show it.
2. Approve and Send back go through the page's normal `updateField`, so every
   existing confirm/handoff behaviour applies.
3. Strings exist in every locale (`parity.test.ts`).
4. No server change.

### Touch points
- `packages/views/workflows/approval-panel.tsx` (+ `.test.tsx`): new.
- `packages/views/issues/components/issue-detail.tsx`: import, and
  `<ApprovalPanel …/>` right before `<div id={descriptionSectionId}>`, after
  `IssueDuplicateBanner`.
- `packages/views/locales/*/issues.json`: `workflows.approval.*`.

### On conflict
If the issue page layout moves, mount the panel at the top of the main reading
column, after any banners and before the title. Re-add the locale keys under
`workflows` (or wherever workflow strings move to) for every locale.

### Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run workflows/approval-panel.test.tsx locales`

## Part: old patch 0005 — workflow-by-type (F3, server) · design D6

### Requirement
In a project, rules map an option of one select property (e.g. Type) to a workflow.
An issue whose property has that option follows that workflow; anything else follows
the project's workflow, then the workspace Default. Reference: Jira workflow schemes.

- Table `project_workflow_rule (project_id, workspace_id, property_id, option_id,
  workflow_id, position)`, PK (project_id, property_id, option_id); index on
  (workspace_id, workflow_id) CONCURRENTLY. No foreign keys; workspace deletion
  deletes the rows (`deleted_project_workflow_rules`), manifest lists the table.
- `issueworkflow.ForIssue(ctx, q, ws, projectID, propertiesJSON)` and `RuleWorkflow`;
  `Querier` gains `ListProjectWorkflowRules`.
- Every former `ForProject` caller that has an issue uses `ForIssue`: issue update
  (`applyIssueWorkflow`, prev.Properties), create (request properties), service create
  (`createWorkflow`, p.Properties), claim brief (`claimProjectWorkflow`), gate check
  (`refuseAgentLeavingGate`, patch 0001), PR automation (`pr_auto_complete.go`, v0.7.2+). The
  comment-trigger preview passes no properties.
- API `GET/PUT /api/projects/{id}/workflow-rules` inside the projects route group;
  PUT owner/admin; validates select property, option and workflow; no duplicate options.
- `SetIssueProperty`: an agent changing a rule property while the issue sits in a
  person-handled step gets 403 `owner_gate`; after a change, a status the new
  workflow lacks moves to `MoveTarget` (no handoff), published as `issue:updated`.

### Invariants
1. No rules = upstream behaviour (ForIssue falls back to ForProjectRow).
2. Existing issues keep their status when rules change; they follow the new workflow
   on their next move or property change.
3. Add-only schema.

### On conflict
New ForProject callers upstream: switch those that have an issue to ForIssue. If
upstream adds per-issue or per-type workflows, compare and drop this patch.

### Tests
`cd server && go run ./cmd/migrate up && go test ./internal/handler/ -run 'WorkflowRule|FollowsRuleWorkflow|ChangingType|WorkspaceDeletion' -count=1`

## Part: old patch 0006 — workflow-by-type-ui (F3, UI) · design D6

Needs 0005.

### Requirement
Project sidebar row "By Type" under "Workflow": a summary ("Copy → content, …" or "Same
for every Type"); owners/admins open a popover to choose the select property and, per
option, the workflow (or "Project workflow"); Save replaces the rules.
Core: `@multica/core/workflow-rules` (query + mutation), client methods
`listProjectWorkflowRules`, `setProjectWorkflowRules`. Strings `issues.workflows.rules.*`
in every locale.

Deviation from D6: a sidebar popover instead of a settings page, because Multica keeps
project settings in the project's sidebar (Workflow row).

### On conflict
Mount next to `ProjectWorkflowPropRow` in `project-detail.tsx`; re-add the client block
before the goals block.

### Tests
`pnpm install --frozen-lockfile --ignore-scripts && (cd packages/views && npx vitest run locales projects workflows)`

## Part: old patch 0007 — board-columns (F4) · design D3, D3b, D3c, D3d

Needs 0005/0006 (workflow rules).

### Requirement
A project board's columns, when the project has workflow rules (by Type), are the
project workflow's steps plus each rule workflow's middle steps it lacks, inserted
before the closing steps (blocked, done, cancelled). This is design D3d "Automatic"
for a board with several workflows. One workflow: unchanged.

### What the rest of D3–D3d uses instead (no patch, Multica already has it)
Following "best practice first / reuse before inventing":
- Saved boards per team (D3 tabs, D3b "Tech team") = Multica saved views scoped to the
  project (board mode, filters by Domain/Type), pinned to the sidebar; created from
  packages by `opc.mjs` (`views:`).
- Swimlanes (D3c) = Multica's swimlane mode, grouping by assignee/project/property.
- Quick filters = Multica's filter chips (property filters for Type, Domain, Goal).

### Deviations still open (need the owner)
- "Awaiting approval" lifecycle column and days-in-column badge (D3) are not built.
- Custom column mapping (D3d Custom) is not built.

### Touch points
`packages/core/workflow-rules/index.ts` (`mergeRuleWorkflows`, plain options object),
`packages/views/issues/utils/workflow-columns.ts` (`useColumnWorkflow`), its test mocks
`useQuery`.

### Tests
`pnpm install --frozen-lockfile --ignore-scripts && (cd packages/core && npx vitest run workflow-rules) && (cd packages/views && npx vitest run issues/utils workflows)`

## Part: old patch 0015 — approval-pr (F1) · design D5b

Needs 0002 (approval-panel).

### Requirement
In the approval panel, list the issue's pull requests: those linked to the issue
(upstream GitHub integration), then GitHub PR links written in its description and
comments (agents post "Approve PR <link>"), newest mention first, one per PR, up to
three. Each row: repo#number (· title, state when linked) and **Approve on GitHub**,
opening the PR's `/files` review page in a new tab, plus the note "Approve the PR on
GitHub with your account, then Approve here." Approve itself is unchanged (moves to
the next step, e.g. `release`). Reference: Jira's development panel on approvals.

No server change; approving on GitHub stays with the owner's own GitHub account.

### Touch points
`packages/views/workflows/approval-prs.tsx` (+ test), one import and
`<ApprovalPullRequests>` after the gate note in `approval-panel.tsx`, its test mocks the
new module, `workflows.approval.{prs_label,approve_on_github,pr_then_approve}` in
every locale.

### Tests
See series.toml.

## Part: old patch 0016 — type-first (F3)

Needs 0005/0006 (workflow rules).

### Requirement
In a project whose workflow rules read a property (Type), that property is always shown:
in the create-issue form (manual mode) as a pill next to the other fields, and on the
issue page as a property row, without going through "Add property" / "… → Custom
properties". Other custom properties keep upstream's progressive disclosure.
Reference: Jira's create dialog puts the work type first. Owner (2026-11-10): "không
thấy property type và domain trong issues".

### Touch points
`ruleProperties` in `packages/core/workflow-rules/index.ts` (+ test); in
`packages/views/modals/create-issue.tsx` and `issues/components/issue-detail.tsx` one
rules query and one extra condition in the custom-property row filter.

### Tests
See series.toml.

## Part: old patch 0017 — derived-property (F3)

### Requirement
A select property whose description contains "Derived from <Source>" (Source = another
select property's name) gets its value from the Source option's name, the part before
" · ": Type "Tech · Feature" → Domain "Tech". Applied on issue create and on every set of
a property (`PUT /api/issues/{id}/properties/{pid}`). No matching option → the derived
value is left as it is. Owner decision (2026-11-10): pick Type, Domain fills itself.

### Invariants
No migration; the rule lives in the property description, so it is visible on the
property page and declared by packages. Without such a description nothing changes.

### Touch points
`server/internal/service/vantt_derive.go`; three lines after property validation in
`service/issue.go` (create); a block after the value write in `handler/property.go`
`SetIssueProperty`; test `handler/vantt_derive_test.go`.

### Tests
See series.toml.

## Part: old patch 0018 — send-back-limit (F1, server)

### Requirement
A workflow may cap how many times **agents** send an issue back along a step's
back edge (`back_status_key`, e.g. review → build) with **no person moving the
issue in between**. One more agent send-back past the cap gets
`409 {"code":"send_back_limit","status","back","count","limit","error"}`; the
batch endpoint skips that issue for the agent. The error tells the agent to
leave the issue where it is, comment the dispute and @mention a person.

- `GET /api/issue-workflows/{id}/limits` → `{"max_send_backs": n | null}` (any member).
- `PUT /api/issue-workflows/{id}/limits` with `{"max_send_backs": n}`; `0` or
  `null` clears it. Same people who may edit the workflow.
- Stored in `vantt_workflow_limit (workflow_id PK, max_send_backs)`, not in
  `issue_workflow.steps`: the workflow editor rebuilds steps and would drop an
  unknown field.

Why: two agents that disagree (reviewer ↔ coder) otherwise bounce an issue
forever; the only stop was a sentence in the instructions. Same idea as
upstream's wake-up loop guard ("rules waking each other without a person in
between"), and as fgOPC's workflow rule `max_send_backs`.

### Invariants
1. Members are never refused, and a member's status move starts a fresh count.
2. Only agent moves count, and only moves `from → to` where `to` is the back
   edge of `from` in the issue's **current** workflow.
3. Forward moves and non-status fields are never refused.
4. No row for the workflow = upstream behaviour.
5. Runs inside `refuseAgentLeavingGate` (patch 0001), after the owner-gate
   check, only for a status change without a project move.

### Touch points
- `server/migrations/9009_vantt_workflow_limit.{up,down}.sql`: new table.
- `server/internal/handler/vantt_send_back_limit.go`: error, count, API.
- `server/internal/handler/issue_workflow_entry.go`: call from
  `refuseAgentLeavingGate`; `writeIssueWorkflowError` maps the error to 409.
- `server/cmd/server/router.go`: `/limits` routes under `/api/issue-workflows/{id}`.
- `server/pkg/db/queries/workspace_delete.sql` + generated: delete limits with
  the workspace (by the workflow's workspace); manifest test classifies the table.
- Counting reads `activity_log` rows `status_changed` (`details.from/to`,
  `actor_type`), written asynchronously by the activity listener.

### On conflict
Keep the check wherever patch 0001's gate check lives. If upstream adds its own
loop/budget guard for workflow back edges, compare and prefer theirs. A deleted
workflow leaves an orphan limit row keyed by a UUID that is never reused;
harmless.

### Tests
`cd server && go test ./internal/handler/ -run 'SendBack|OwnerGate|WorkspaceDeletionManifest' -count=1`
(needs `DATABASE_URL` to a migrated test database).
