-- vantt workflow-by-type (patch workflow-by-type, F3): in a project, an issue
-- whose select property (e.g. Type) has a given option follows another workflow.
-- No foreign keys: the handler validates project, property, option and workflow.
CREATE TABLE IF NOT EXISTS project_workflow_rule (
    project_id   UUID NOT NULL,
    workspace_id UUID NOT NULL,
    property_id  UUID NOT NULL,
    option_id    TEXT NOT NULL,
    workflow_id  UUID NOT NULL,
    position     INT  NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, property_id, option_id)
);
