-- vantt goals (patch goals, F2).

-- name: ListGoals :many
SELECT * FROM goal
WHERE workspace_id = $1
ORDER BY position ASC, created_at ASC;

-- name: GetGoalInWorkspace :one
SELECT * FROM goal
WHERE id = $1 AND workspace_id = $2;

-- name: CreateGoal :one
INSERT INTO goal (
    workspace_id, parent_id, title, description, unit, direction,
    start_value, target_value, current_value, due_date, status, measured_by,
    owner_type, owner_id, position, source_ref
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: UpdateGoal :one
UPDATE goal SET
    parent_id    = $3,
    title        = $4,
    description  = $5,
    unit         = $6,
    direction    = $7,
    start_value  = $8,
    target_value = $9,
    due_date     = $10,
    status       = $11,
    measured_by  = $12,
    owner_type   = $13,
    owner_id     = $14,
    position     = $15,
    source_ref   = $16,
    pending      = $17,
    updated_at   = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: SetGoalCurrent :one
UPDATE goal SET
    current_value      = $3,
    current_updated_at = now(),
    updated_at         = now()
WHERE id = $1 AND workspace_id = $2
RETURNING *;

-- name: DeleteGoal :exec
DELETE FROM goal WHERE id = $1 AND workspace_id = $2;

-- name: CountGoalChildren :one
SELECT count(*) FROM goal WHERE parent_id = $1 AND workspace_id = $2;

-- name: CreateGoalValue :one
INSERT INTO goal_value (goal_id, workspace_id, value, note, actor_type, actor_id, synced, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE(sqlc.narg('recorded_at')::timestamptz, now()))
RETURNING *;

-- name: MarkGoalValuesSynced :exec
UPDATE goal_value SET synced = true
WHERE goal_id = $1 AND workspace_id = $2 AND id = ANY(sqlc.arg('ids')::uuid[]);

-- name: ListGoalValues :many
SELECT * FROM goal_value
WHERE goal_id = $1 AND workspace_id = $2
ORDER BY created_at ASC, id ASC;

-- name: DeleteGoalValues :exec
DELETE FROM goal_value WHERE goal_id = $1 AND workspace_id = $2;

-- name: ListGoalLinks :many
SELECT * FROM goal_link
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: DeleteGoalLinks :exec
DELETE FROM goal_link WHERE goal_id = $1 AND workspace_id = $2;

-- name: CreateGoalLink :exec
INSERT INTO goal_link (goal_id, workspace_id, target_type, target_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;
