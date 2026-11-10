-- vantt workflow-by-type (patch workflow-by-type, F3).

-- name: ListProjectWorkflowRules :many
SELECT * FROM project_workflow_rule
WHERE project_id = $1 AND workspace_id = $2
ORDER BY position ASC, created_at ASC;

-- name: DeleteProjectWorkflowRules :exec
DELETE FROM project_workflow_rule WHERE project_id = $1 AND workspace_id = $2;

-- name: CreateProjectWorkflowRule :exec
INSERT INTO project_workflow_rule (project_id, workspace_id, property_id, option_id, workflow_id, position)
VALUES ($1, $2, $3, $4, $5, $6);
