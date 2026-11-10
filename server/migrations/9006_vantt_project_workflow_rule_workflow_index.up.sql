CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_project_workflow_rule_workflow ON project_workflow_rule (workspace_id, workflow_id);
