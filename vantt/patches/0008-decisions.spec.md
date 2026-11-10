# Patch 0008 — decisions (F5, server) · design D4

## Requirement
`issue_decision` (add-only, no foreign keys; deleted with the workspace): question,
2–8 options (`id`, `label`, `description`, at most one `recommended`), due date, asked by
(member or agent, from the task token), status open/answered/cancelled, answer option,
note, who answered and when.
API: `GET/POST /api/issues/{id}/decisions`, `POST …/decisions/{decisionId}/answer`
(people only; 409 when not open), `POST …/cancel`, `GET /api/issues/decisions/open`.
Waiting is the workflow's job: the issue sits in a person-handled step (patch 0001).
Clients post the answer as a comment, so existing comment triggers wake the agent.
Reference: Jira approvals; Paperclip decisions.

## On conflict
Re-register the routes in the issue route group; keep `deleted_issue_decisions` in
`workspace_delete.sql` and the manifest entry.

## Tests
`cd server && go run ./cmd/migrate up && go test ./internal/handler/ -run 'Decision|WorkspaceDeletion' -count=1`
