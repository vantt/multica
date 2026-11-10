import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { ChildStatusBar, DaysInStatus } from "./child-panel-extras";

const since = (days: number) => new Date(Date.now() - days * 86_400_000 - 60_000).toISOString();
let entries: Record<string, { status: string; since: string }> = {};

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@multica/core/issue-statuses/hooks", () => ({
  useIssueStatuses: () => ({
    categoryOf: (s: string) => (s === "done" ? "done" : s === "todo" ? "unstarted" : "started"),
    colorOf: () => "#888888",
  }),
}));
vi.mock("../utils/status-label", () => ({ useStatusLabel: () => (s: string) => s }));
vi.mock("@tanstack/react-query", () => ({ useQuery: () => ({ data: entries }) }));

afterEach(cleanup);

describe("ChildStatusBar", () => {
  it("summarises children by status, done first", () => {
    renderWithI18n(<ChildStatusBar issues={[{ status: "todo" }, { status: "done" }, { status: "done" }, { status: "in_review" }]} />);
    expect(screen.getByTitle("2 done, 1 in_review, 1 todo")).toBeTruthy();
  });
  it("draws nothing without children", () => {
    const { container } = renderWithI18n(<ChildStatusBar issues={[]} />);
    expect(container.textContent).toBe("");
  });
});

describe("DaysInStatus", () => {
  it("shows whole days in the current status", () => {
    entries = { a: { status: "in_review", since: since(6) } };
    renderWithI18n(<DaysInStatus issue={{ id: "a", status: "in_review", project_id: "p" }} />);
    expect(screen.getByText("6d")).toBeTruthy();
    expect(screen.getByTitle("6 days in in_review")).toBeTruthy();
  });
  it("hides under a day and after the issue moved", () => {
    entries = { a: { status: "todo", since: since(0) }, b: { status: "todo", since: since(3) } };
    renderWithI18n(<DaysInStatus issue={{ id: "a", status: "todo", project_id: "p" }} />);
    renderWithI18n(<DaysInStatus issue={{ id: "b", status: "in_review", project_id: "p" }} />);
    expect(screen.queryByText(/\dd$/)).toBeNull();
  });
});
