-- vantt goals (patch goals, F2): results with a number and a deadline that work
-- moves. No foreign keys: relationships are validated in the handler.
CREATE TABLE IF NOT EXISTS goal (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id       UUID NOT NULL,
    parent_id          UUID,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    unit               TEXT NOT NULL DEFAULT '',
    direction          TEXT NOT NULL DEFAULT 'increase' CHECK (direction IN ('increase', 'decrease')),
    start_value        DOUBLE PRECISION NOT NULL DEFAULT 0,
    target_value       DOUBLE PRECISION NOT NULL,
    current_value      DOUBLE PRECISION NOT NULL DEFAULT 0,
    due_date           DATE,
    status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('planned', 'active', 'achieved', 'missed', 'cancelled')),
    measured_by        TEXT NOT NULL DEFAULT 'manual' CHECK (measured_by IN ('manual', 'script')),
    owner_type         TEXT,
    owner_id           UUID,
    position           DOUBLE PRECISION NOT NULL DEFAULT 0,
    current_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS goal_value (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id      UUID NOT NULL,
    workspace_id UUID NOT NULL,
    value        DOUBLE PRECISION NOT NULL,
    note         TEXT NOT NULL DEFAULT '',
    actor_type   TEXT NOT NULL,
    actor_id     UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS goal_link (
    goal_id      UUID NOT NULL,
    workspace_id UUID NOT NULL,
    target_type  TEXT NOT NULL CHECK (target_type IN ('project', 'issue')),
    target_id    UUID NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (goal_id, target_type, target_id)
);
