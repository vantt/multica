import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Project } from "@multica/core/types";
import { renderWithI18n } from "../test/i18n";
import { WorkflowRulesControl } from "./workflow-rules-row";

const state = vi.hoisted(() => ({ rules: [] as { property_id: string; option_id: string; workflow_id: string }[], admin: false }));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@tanstack/react-query", () => ({
  useQuery: (o: { queryKey: readonly unknown[] }) =>
    o.queryKey[0] === "workflow-rules"
      ? { data: state.rules }
      : { data: [{ id: "prop-1", name: "Type", type: "select", archived: false, config: { options: [{ id: "opt-copy", name: "Copy" }] } }] },
}));
vi.mock("@multica/core/workflow-rules", () => ({
  workflowRulesOptions: () => ({ queryKey: ["workflow-rules"] }),
  useSetWorkflowRules: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@multica/core/properties/queries", () => ({ propertyListOptions: () => ({ queryKey: ["properties"] }) }));
vi.mock("@multica/core/issue-workflows", () => ({ useIssueWorkflows: () => ({ workflows: [{ id: "wf-content", name: "content" }] }) }));
vi.mock("./permissions", () => ({ useWorkflowPermissions: () => ({ canManageProject: state.admin, isAdmin: state.admin }) }));

afterEach(() => {
  cleanup();
  state.rules = [];
});

describe("WorkflowRulesControl", () => {
  it("says every Type is the same when there are no rules", () => {
    renderWithI18n(<WorkflowRulesControl project={{ id: "p-1" } as Project} />);
    expect(screen.getByText("Same for every Type")).toBeTruthy();
  });

  it("summarises rules as option → workflow", () => {
    state.rules = [{ property_id: "prop-1", option_id: "opt-copy", workflow_id: "wf-content" }];
    renderWithI18n(<WorkflowRulesControl project={{ id: "p-1" } as Project} />);
    expect(screen.getByText("Copy → content")).toBeTruthy();
  });
});
