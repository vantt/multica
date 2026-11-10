CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_issue_decision_issue ON issue_decision (workspace_id, issue_id, created_at);
