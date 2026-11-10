import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Issue } from "@multica/core/types";
import { renderWithI18n } from "../test/i18n";
import { DecisionPanel } from "./decision-panel";

const state = vi.hoisted(() => ({ mutate: vi.fn(), decisions: [] as unknown[] }));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@multica/core/auth", () => ({ useAuthStore: (s: (x: { user: { id: string } }) => unknown) => s({ user: { id: "owner-1" } }) }));
vi.mock("@tanstack/react-query", () => ({ useQuery: () => ({ data: state.decisions }) }));
vi.mock("@multica/core/decisions", () => ({
  issueDecisionsOptions: () => ({ queryKey: ["decisions"] }),
  useAnswerDecision: () => ({ mutate: state.mutate, isPending: false }),
}));

const decision = {
  id: "d-1", issue_id: "i-1", question: "Which payment provider?", status: "open", due_date: "2026-11-14",
  asked_by_type: "agent", asked_by_id: null, answer_option_id: null, answer_note: "", answered_by: null, answered_at: null, created_at: "",
  options: [
    { id: "o-polar", label: "Polar", description: "VND payout", recommended: true },
    { id: "o-dodo", label: "Dodo", description: "", recommended: false },
  ],
};

afterEach(() => {
  cleanup();
  state.mutate.mockReset();
});

describe("DecisionPanel", () => {
  it("preselects the recommended option and answers with the picked one", () => {
    state.decisions = [decision];
    renderWithI18n(<DecisionPanel issue={{ id: "i-1" } as Issue} />);
    expect(screen.getByText("Recommended")).toBeTruthy();
    fireEvent.click(screen.getByRole("radio", { name: /Dodo/ }));
    fireEvent.click(screen.getByRole("button", { name: "Answer: Dodo" }));
    expect(state.mutate).toHaveBeenCalledWith({ decision, optionId: "o-dodo", note: "" });
  });

  it("shows nothing without open decisions", () => {
    state.decisions = [{ ...decision, status: "answered" }];
    renderWithI18n(<DecisionPanel issue={{ id: "i-1" } as Issue} />);
    expect(screen.queryByText("Which payment provider?")).toBeNull();
  });
});
