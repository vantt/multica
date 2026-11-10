import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Issue, IssueWorkflow } from "@multica/core/types";
import { renderWithI18n } from "../test/i18n";
import { ApprovalPanel } from "./approval-panel";

const state = vi.hoisted(() => ({
  viewer: "owner-1",
  workflow: null as IssueWorkflow | null,
  comment: vi.fn(),
}));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/auth", () => ({
  useAuthStore: (select: (s: { user: { id: string } }) => unknown) => select({ user: { id: state.viewer } }),
}));
vi.mock("@multica/core/issues/mutations", () => ({
  useCreateComment: () => ({
    mutate: (vars: { content: string }, opts?: { onSuccess?: () => void }) => {
      state.comment(vars.content);
      opts?.onSuccess?.();
    },
    isPending: false,
  }),
}));
vi.mock("@multica/core/issue-workflows", async (importOriginal) => {
  const real = await importOriginal<typeof import("@multica/core/issue-workflows")>();
  return {
    workflowStep: real.workflowStep,
    workflowNextStatus: real.workflowNextStatus,
    useProjectWithWorkflow: () => ({ project: { lead_type: null, lead_id: null }, workflow: state.workflow }),
  };
});
vi.mock("../issues/utils/status-label", () => ({ useStatusLabel: () => (key: string) => key }));

function delivery(): IssueWorkflow {
  return {
    id: "wf-1",
    workspace_id: "workspace-1",
    name: "delivery",
    description: "",
    is_default: false,
    owner_id: null,
    steps: [
      { status_key: "todo", handler: { type: "agent", id: "agent-1" }, instructions: "" },
      {
        status_key: "in_review",
        handler: { type: "member", id: "owner-1" },
        instructions: "",
        next_status_key: "release",
        back_status_key: "build",
      },
      { status_key: "release", handler: { type: "agent", id: "agent-2" }, instructions: "" },
    ],
    project_ids: ["project-1"],
    created_at: "",
    updated_at: "",
  };
}

const issue = (status: string) =>
  ({ id: "issue-1", status, project_id: "project-1", creator_type: "agent", creator_id: "agent-1" }) as Issue;

beforeEach(() => {
  state.viewer = "owner-1";
  state.workflow = delivery();
  state.comment.mockReset();
});
afterEach(cleanup);

describe("ApprovalPanel", () => {
  it("approves into the step's next status", () => {
    const onUpdate = vi.fn();
    renderWithI18n(<ApprovalPanel issue={issue("in_review")} onUpdateField={onUpdate} />);
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    expect(onUpdate).toHaveBeenCalledWith({ status: "release" });
  });

  it("sends back with a reason as a comment, into the step's back status", () => {
    const onUpdate = vi.fn();
    renderWithI18n(<ApprovalPanel issue={issue("in_review")} onUpdateField={onUpdate} />);
    fireEvent.click(screen.getByRole("button", { name: "Send back" }));
    const submit = screen.getByRole("button", { name: "Send back to build" });
    expect((submit as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("What needs to change?"), { target: { value: "Allow two machines" } });
    fireEvent.click(submit);
    expect(state.comment).toHaveBeenCalledWith("Allow two machines");
    expect(onUpdate).toHaveBeenCalledWith({ status: "build" });
  });

  it("stays hidden for anyone but the step's person, and on agent steps", () => {
    state.viewer = "someone-else";
    const { unmount } = renderWithI18n(<ApprovalPanel issue={issue("in_review")} onUpdateField={vi.fn()} />);
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    unmount();
    state.viewer = "owner-1";
    renderWithI18n(<ApprovalPanel issue={issue("todo")} onUpdateField={vi.fn()} />);
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
  });
});
