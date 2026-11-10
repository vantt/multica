-- vantt decisions (patch decisions, F5).

-- name: CreateIssueDecision :one
INSERT INTO issue_decision (issue_id, workspace_id, question, options, due_date, asked_by_type, asked_by_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListIssueDecisions :many
SELECT * FROM issue_decision
WHERE issue_id = $1 AND workspace_id = $2
ORDER BY created_at ASC;

-- name: ListOpenIssueDecisions :many
SELECT * FROM issue_decision
WHERE workspace_id = $1 AND status = 'open'
ORDER BY created_at ASC;

-- name: GetIssueDecision :one
SELECT * FROM issue_decision
WHERE id = $1 AND workspace_id = $2;

-- name: AnswerIssueDecision :one
UPDATE issue_decision SET
    status = 'answered',
    answer_option_id = $3,
    answer_note = $4,
    answered_by = $5,
    answered_at = now()
WHERE id = $1 AND workspace_id = $2 AND status = 'open'
RETURNING *;

-- name: CancelIssueDecision :one
UPDATE issue_decision SET status = 'cancelled'
WHERE id = $1 AND workspace_id = $2 AND status = 'open'
RETURNING *;
