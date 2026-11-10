// @vitest-environment node
import { describe, expect, it } from "vitest";
import { mergeRuleWorkflows } from "./index";

const wf = (id: string, keys: string[]) => ({ id, steps: keys.map((status_key) => ({ status_key })) });

describe("mergeRuleWorkflows", () => {
  it("adds rule workflows' middle steps before the closing steps", () => {
    const delivery = wf("d", ["backlog", "todo", "build", "in_review", "blocked", "done", "cancelled"]);
    const content = wf("c", ["todo", "content_review", "in_review", "done", "cancelled"]);
    const merged = mergeRuleWorkflows(delivery, [{ property_id: "p", option_id: "o", workflow_id: "c" }], [delivery, content], ["blocked", "done", "cancelled"]);
    expect(merged?.steps.map((s) => s.status_key)).toEqual(["backlog", "todo", "build", "in_review", "content_review", "blocked", "done", "cancelled"]);
  });

  it("keeps the project workflow without rules", () => {
    const delivery = wf("d", ["todo", "done"]);
    expect(mergeRuleWorkflows(delivery, [], [delivery], ["done"])).toBe(delivery);
  });
});
