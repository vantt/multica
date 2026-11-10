# Patch 0018 — send-back-limit (F1, server)

## Requirement
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

## Invariants
1. Members are never refused, and a member's status move starts a fresh count.
2. Only agent moves count, and only moves `from → to` where `to` is the back
   edge of `from` in the issue's **current** workflow.
3. Forward moves and non-status fields are never refused.
4. No row for the workflow = upstream behaviour.
5. Runs inside `refuseAgentLeavingGate` (patch 0001), after the owner-gate
   check, only for a status change without a project move.

## Touch points
- `server/migrations/9009_vantt_workflow_limit.{up,down}.sql`: new table.
- `server/internal/handler/vantt_send_back_limit.go`: error, count, API.
- `server/internal/handler/issue_workflow_entry.go`: call from
  `refuseAgentLeavingGate`; `writeIssueWorkflowError` maps the error to 409.
- `server/cmd/server/router.go`: `/limits` routes under `/api/issue-workflows/{id}`.
- `server/pkg/db/queries/workspace_delete.sql` + generated: delete limits with
  the workspace (by the workflow's workspace); manifest test classifies the table.
- Counting reads `activity_log` rows `status_changed` (`details.from/to`,
  `actor_type`), written asynchronously by the activity listener.

## On conflict
Keep the check wherever patch 0001's gate check lives. If upstream adds its own
loop/budget guard for workflow back edges, compare and prefer theirs. A deleted
workflow leaves an orphan limit row keyed by a UUID that is never reused;
harmless.

## Tests
`cd server && go test ./internal/handler/ -run 'SendBack|OwnerGate|WorkspaceDeletionManifest' -count=1`
(needs `DATABASE_URL` to a migrated test database).
