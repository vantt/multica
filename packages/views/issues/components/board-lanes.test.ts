import { describe, expect, it, vi } from "vitest";
import { buildColumns } from "../utils/drag-utils";
import { buildLaneGroups, lanePages, resolveLaneGroup } from "./board-lanes";

vi.mock("@multica/core/issue-workflows", () => ({
  useIssueWorkflows: () => ({ workflows: [] }),
  useProjectWorkflow: () => null,
  workflowAllowsStatus: (wf: { steps: { status_key: string }[] }, s: string) => wf.steps.some((x) => x.status_key === s),
}));

const category = (s: string) =>
  (({ backlog: "unstarted", todo: "unstarted", done: "done", cancelled: "closed" }) as Record<string, string>)[s] ?? "started";
const visible = ["backlog", "todo", "design", "spec_approval", "build", "in_review", "content_review", "blocked", "done", "cancelled"];
const groups = buildLaneGroups(visible, category as never, ["spec_approval", "in_review"], (l) => l);

describe("buildLaneGroups", () => {
  it("folds statuses into four lanes, approvals in Awaiting approval", () => {
    expect(groups.map((g) => [g.id, g.statuses])).toEqual([
      ["lane:todo", ["backlog", "todo"]],
      ["lane:in_progress", ["design", "build", "content_review", "blocked"]],
      ["lane:awaiting_approval", ["spec_approval", "in_review"]],
      ["lane:done", ["done", "cancelled"]],
    ]);
  });
  it("puts cards in the lane of their status", () => {
    const issues = [{ id: "a", status: "in_review" }, { id: "b", status: "build" }, { id: "c", status: "spec_approval" }];
    const cols = buildColumns(issues as never, groups, "status");
    expect(cols["lane:awaiting_approval"]).toEqual(["a", "c"]);
    expect(cols["lane:in_progress"]).toEqual(["b"]);
  });
});

describe("resolveLaneGroup", () => {
  const awaiting = groups[2]!;
  const content = { steps: [{ status_key: "todo" }, { status_key: "content_review" }, { status_key: "in_review" }] };
  it("keeps the card's status inside its lane", () => {
    expect(resolveLaneGroup(awaiting, { status: "in_review" }, content as never)?.status).toBe("in_review");
  });
  it("takes the lane's first status the card's workflow allows", () => {
    expect(resolveLaneGroup(awaiting, { status: "content_review" }, content as never)?.status).toBe("in_review");
  });
  it("refuses a lane the workflow has no step in", () => {
    expect(resolveLaneGroup(groups[0]!, { status: "in_review" }, { steps: [{ status_key: "in_review" }] } as never)).toBeNull();
  });
  it("leaves ordinary columns alone", () => {
    const g = { id: "status:todo", title: "todo", status: "todo" };
    expect(resolveLaneGroup(g, { status: "done" }, null)).toBe(g);
  });
});

describe("lanePages", () => {
  it("merges the pages of a lane's statuses", () => {
    const more = vi.fn();
    const page = (total: number, hasMore = false) => ({ total, loaded: total, hasMore, isLoading: false, isFetching: false, isError: false, loadMore: more, retry: vi.fn() });
    const pages = lanePages(groups, { spec_approval: page(2, true), in_review: page(3) });
    expect(pages["lane:awaiting_approval"]?.total).toBe(5);
    pages["lane:awaiting_approval"]!.loadMore();
    expect(more).toHaveBeenCalledTimes(1);
  });
});
