# Patch 0001 — owner-gates (F1, server)

## Requirement
A workflow step whose handler resolves to a **member** (a person: `member`, or
`project_lead`/`creator` when that resolves to a member) is an **owner gate**.
An **agent** actor (request carries a task token: `X-Actor-Source: task_token`)
may not move an issue that sits in a gate **out** of it:
- changing its status → `403 {"code":"owner_gate","status":"<current>","error":"…"}`;
- moving it to another project → same 403 (a project move remaps the status);
- the batch endpoint skips such an issue for that agent and applies the others.

References: Jira workflow conditions (only some users may perform a transition);
Paperclip refuses a review move without an approver.

## Invariants
1. Members are never affected.
2. Agents still move issues **into** a gate (that is how work reaches the owner) and
   change any other field of a gated issue (title, description, labels…).
3. Steps handled by an agent, a squad, or with no handler are not gates.
4. Without member-handled steps the behaviour equals upstream.
5. The check runs before `applyIssueWorkflow`, using the issue's **current**
   project workflow and status (`prev`), not the requested ones.

## Touch points
- `server/internal/issueworkflow/issueworkflow.go`: `GateError`, `PersonGate`
  (placed before `MoveTarget`).
- `server/internal/handler/issue_workflow_entry.go`: `writeIssueWorkflowError`
  maps `GateError` → 403; `refuseAgentLeavingGate` (before `agentRunOnIssue`).
- `server/internal/handler/issue.go`: `UpdateIssue` and `BatchUpdateIssues` call
  `refuseAgentLeavingGate` right after `callerType` is resolved, before
  `applyIssueWorkflow`.
- `server/internal/handler/vantt_owner_gates_test.go`: new tests.

## On conflict
Upstream moves or renames the workflow write path: put the gate check wherever
the caller type is resolved and before the workflow is applied, for every write
path that can change `status` or `project_id` (single update, batch update, and
any new one). Keep the 403 code `owner_gate`. If upstream adds its own
role/permission gating for workflow steps, compare and prefer theirs; mark this
patch `merged`/drop it when it covers invariants 1–4.

## Tests
`cd server && go test ./internal/handler/ -run 'OwnerGate|AgentStepIsNotAGate' -count=1`
(needs `DATABASE_URL` to a migrated test database).
