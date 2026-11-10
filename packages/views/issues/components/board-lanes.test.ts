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
  it("folds statuses into five lanes, approvals in Awaiting approval", () => {
    expect(groups.map((g) => [g.id, g.statuses])).toEqual([
      ["lane:backlog", ["backlog"]],
      ["lane:todo", ["todo"]],
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
  const awaiting = groups[3]!;
  const content = { steps: [{ status_key: "todo" }, { status_key: "content_review" }, { status_key: "in_review" }] };
  it("keeps the card's status inside its lane", () => {
    expect(resolveLaneGroup(awaiting, { status: "in_review" }, content as never)?.status).toBe("in_review");
  });
  it("takes the lane's first status the card's workflow allows", () => {
    expect(resolveLaneGroup(awaiting, { status: "content_review" }, content as never)?.status).toBe("in_review");
  });
  it("refuses a lane the workflow has no step in", () => {
    expect(resolveLaneGroup(groups[1]!, { status: "in_review" }, { steps: [{ status_key: "in_review" }] } as never)).toBeNull();
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

describe("oldestFirst and the Awaiting approval lane", () => {
  it("orders approvals by how long they have waited", async () => {
    const { oldestFirst } = await import("./board-lanes");
    const since = { a: { status: "in_review", since: "2026-11-09T00:00:00Z" }, b: { status: "in_review", since: "2026-11-01T00:00:00Z" } };
    expect(oldestFirst(["a", "c", "b"], since)).toEqual(["b", "a", "c"]);
    const g = buildLaneGroups(["in_review", "done"], category as never, ["in_review"], (l) => l, since);
    const cols = buildColumns([{ id: "a", status: "in_review" }, { id: "b", status: "in_review" }] as never, g, "status");
    expect(cols["lane:awaiting_approval"]).toEqual(["b", "a"]);
  });
});

describe("issueGoal", () => {
  it("takes the issue's goal, else its parent's, else its project's", async () => {
    const { issueGoal } = await import("./board-lanes");
    const goal = (id: string, type: string, target: string) => ({ id, title: id, links: [{ target_type: type, target_id: target }] });
    const goals = [goal("G-project", "project", "p1"), goal("G-parent", "issue", "parent1"), goal("G-own", "issue", "i1")] as never;
    expect(issueGoal(goals, { id: "i1", parent_issue_id: "parent1", project_id: "p1" })?.id).toBe("G-own");
    expect(issueGoal(goals, { id: "i2", parent_issue_id: "parent1", project_id: "p1" })?.id).toBe("G-parent");
    expect(issueGoal(goals, { id: "i3", parent_issue_id: null, project_id: "p1" })?.id).toBe("G-project");
    expect(issueGoal(goals, { id: "i4", parent_issue_id: null, project_id: null })).toBeNull();
  });
});
