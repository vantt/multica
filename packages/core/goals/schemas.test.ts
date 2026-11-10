// @vitest-environment node
import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import { EMPTY_GOALS, GoalDetailSchema, ListGoalsSchema } from "./schemas";
import { goalTree, goalsForTarget } from "./queries";
import type { Goal } from "./types";

const goal = (over: Partial<Goal> = {}) =>
  ListGoalsSchema.parse({ goals: [{ id: "g1", title: "Paying customers", target_value: 50, ...over }] }).goals[0]!;

describe("goal schemas", () => {
  it("fills defaults and tolerates unknown enums", () => {
    const g = goal({ health: "sideways" as never, links: "bad" as never });
    expect(g.health).toBe("no_target_date");
    expect(g.links).toEqual([]);
    expect(g.current_value).toBe(0);
  });

  it("falls back on a malformed list", () => {
    expect(parseWithFallback("nope", ListGoalsSchema, EMPTY_GOALS, { endpoint: "test" })).toEqual(EMPTY_GOALS);
  });

  it("parses a detail with history", () => {
    const d = GoalDetailSchema.parse({ goal: { id: "g1", title: "x", target_value: 1 }, history: [{ id: "v1", value: 3 }] });
    expect(d.history[0]?.value).toBe(3);
  });
});

describe("goal helpers", () => {
  it("orders the tree parents first", () => {
    const tree = goalTree([goal({ id: "c", parent_id: "p" }), goal({ id: "p" })]);
    expect(tree.map((t) => [t.goal.id, t.depth])).toEqual([["p", 0], ["c", 1]]);
  });

  it("finds goals for an issue or its parent", () => {
    const g = goal({ links: [{ target_type: "issue", target_id: "epic-1" }] });
    expect(goalsForTarget([g], "issue", ["child-1", "epic-1"])).toHaveLength(1);
    expect(goalsForTarget([g], "issue", ["other"])).toHaveLength(0);
  });
});
