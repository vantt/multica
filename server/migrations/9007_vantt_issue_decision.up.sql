-- vantt decisions (patch decisions, F5): a question on an issue with options for
-- a person to pick, one of them recommended. No foreign keys.
CREATE TABLE IF NOT EXISTS issue_decision (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id         UUID NOT NULL,
    workspace_id     UUID NOT NULL,
    question         TEXT NOT NULL,
    options          JSONB NOT NULL DEFAULT '[]',
    status           TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'answered', 'cancelled')),
    due_date         DATE,
    asked_by_type    TEXT NOT NULL,
    asked_by_id      UUID,
    answer_option_id TEXT,
    answer_note      TEXT NOT NULL DEFAULT '',
    answered_by      UUID,
    answered_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
