-- vantt send-back-limit: how many times agents may send an issue back along a
-- step's back edge, with no person moving it in between, before the server
-- refuses. One row per workflow; no row = no limit (upstream behaviour). Kept
-- out of issue_workflow.steps so the workflow editor, which rebuilds steps,
-- never drops it. No foreign keys.
CREATE TABLE IF NOT EXISTS vantt_workflow_limit (
    workflow_id    UUID PRIMARY KEY,
    max_send_backs INT NOT NULL CHECK (max_send_backs >= 1),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
