# Patch 0004 — decisions

feat(decisions): structured decisions on issues: API, issue panel, CLI

One patch per feature since 2026-11-11 (owner decision). Made from the old patches 0008 decisions, 0009 decisions-ui; each part below is that patch's spec, unchanged. Code comments still name the old numbers (`vantt 0019`, `patch 0020`): the table in `series.toml` maps them.

## Tests (whole patch)
`cd server && go run ./cmd/migrate up >/dev/null && go test ./internal/handler/ -run 'Decision|WorkspaceDeletion' -count=1 && go test ./cmd/multica/ -run 'Decision|IssueAsk' -count=1 && cd .. && pnpm install --frozen-lockfile --ignore-scripts >/dev/null && (cd packages/core && npx vitest run decisions) && (cd packages/views && npx vitest run workflows locales)`

## Part: old patch 0008 — decisions (F5, server) · design D4

### Requirement
`issue_decision` (add-only, no foreign keys; deleted with the workspace): question,
2–8 options (`id`, `label`, `description`, at most one `recommended`), due date, asked by
(member or agent, from the task token), status open/answered/cancelled, answer option,
note, who answered and when.
API: `GET/POST /api/issues/{id}/decisions`, `POST …/decisions/{decisionId}/answer`
(people only; 409 when not open), `POST …/cancel`, `GET /api/issues/decisions/open`.
Waiting is the workflow's job: the issue sits in a person-handled step (patch 0001).
Clients post the answer as a comment, so existing comment triggers wake the agent.
Reference: Jira approvals; Paperclip decisions.

### On conflict
Re-register the routes in the issue route group; keep `deleted_issue_decisions` in
`workspace_delete.sql` and the manifest entry.

### Tests
`cd server && go run ./cmd/migrate up && go test ./internal/handler/ -run 'Decision|WorkspaceDeletion' -count=1`

## Part: old patch 0009 — decisions-ui (F5, UI) · design D4

Needs 0008. `DecisionPanel` above `ApprovalPanel` on the issue page: question, due date,
options as a radio group (recommended marked and preselected), optional note, "Answer:
<option>". `useAnswerDecision` records the answer then posts
`**<question>** → <option>` (+ note) as a comment. Strings `issues.workflows.decision.*`
in every locale. Core module `@multica/core/decisions` (plain query options object).

### Tests
`pnpm install --frozen-lockfile --ignore-scripts && cd packages/views && npx vitest run workflows locales`

## Part: CLI for agents (added 2026-11-11)

### Requirement
Agents reach Multica only through the `multica` CLI (the runtime forbids raw HTTP), so
without a command they could not ask. `server/cmd/multica/vantt_cmd_issue_decision.go`:

- `multica issue ask <issue> --question "…" --option "A" --option "B|why" [--recommended B] [--due YYYY-MM-DD] [--output json]`
  → `POST /api/issues/{id}/decisions`. Checks 2–8 options, labels, the recommended label
  and the date before sending.
- `multica issue decision list <issue>` / `--open` → the issue's decisions / every open one.
- `multica issue decision cancel <issue> <decision-id>`.
- No answer command: the server lets only people answer.

### Invariants
- The file's `init` sets the help templates itself: `main.go` applies them before this
  file's `init` runs.

### On conflict
Re-register `issueAskCmd` and `issueDecisionCmd` under `issueCmd`; helpers used:
`newAPIClient`, `resolveIssueRef`, `strVal`, `displayID`, `cli.PrintTable/PrintJSON`.

### Tests
`cd server && go test ./cmd/multica/ -run 'Decision|IssueAsk' -count=1`

## Part: the answer wakes the agent that asked (added 2026-11-11)

### Requirement
An agent that asks usually moves the issue to `blocked` and hands it to the person, so a
plain answer comment on an issue assigned to a member woke nobody (seen on STU-47).
The answer comment now ends with a mention of the asking agent,
`[@<name>](mention://agent/<asked_by_id>)` (brackets in the name escaped); a mention
enqueues that agent whoever the issue is assigned to. Built by
`decisionAnswerComment` in `@multica/core/decisions` (web panel); the Telegram button
builds the same text server side (patch 0006, `Handler.decisionAnswerComment`).
A person's question gets no mention.

### Tests
`cd packages/core && npx vitest run decisions`
