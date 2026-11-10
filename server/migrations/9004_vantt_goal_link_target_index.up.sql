CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_goal_link_target ON goal_link (workspace_id, target_type, target_id);
