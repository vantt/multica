# Patch 0005 — workflow-by-type (F3, server) · design D6

## Requirement
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
  (`refuseAgentLeavingGate`, patch 0001). The comment-trigger preview passes no properties.
- API `GET/PUT /api/projects/{id}/workflow-rules` inside the projects route group;
  PUT owner/admin; validates select property, option and workflow; no duplicate options.
- `SetIssueProperty`: an agent changing a rule property while the issue sits in a
  person-handled step gets 403 `owner_gate`; after a change, a status the new
  workflow lacks moves to `MoveTarget` (no handoff), published as `issue:updated`.

## Invariants
1. No rules = upstream behaviour (ForIssue falls back to ForProjectRow).
2. Existing issues keep their status when rules change; they follow the new workflow
   on their next move or property change.
3. Add-only schema.

## On conflict
New ForProject callers upstream: switch those that have an issue to ForIssue. If
upstream adds per-issue or per-type workflows, compare and drop this patch.

## Tests
`cd server && go run ./cmd/migrate up && go test ./internal/handler/ -run 'WorkflowRule|FollowsRuleWorkflow|ChangingType|WorkspaceDeletion' -count=1`
