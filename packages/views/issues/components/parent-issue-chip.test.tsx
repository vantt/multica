import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { ParentIssueChip } from "./parent-issue-chip";

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("@multica/core/issues/queries", () => ({ issueDetailOptions: (_ws: string, id: string) => ({ queryKey: ["issue", id] }) }));
vi.mock("@tanstack/react-query", () => ({
  useQuery: (o: { queryKey: unknown[]; enabled?: boolean }) =>
    o.enabled === false ? { data: undefined } : { data: { id: "p-1", identifier: "MOM-30", title: "E8 Landing v3" } },
}));

afterEach(cleanup);

describe("ParentIssueChip", () => {
  it("names the parent of a sub-issue", () => {
    renderWithI18n(<ParentIssueChip parentId="p-1" />);
    expect(screen.getByText("MOM-30")).toBeTruthy();
    expect(screen.getByTitle("Sub-issue of MOM-30 · E8 Landing v3")).toBeTruthy();
  });

  it("shows nothing for a top-level issue", () => {
    renderWithI18n(<ParentIssueChip parentId={null} />);
    expect(screen.queryByText("MOM-30")).toBeNull();
  });
});
