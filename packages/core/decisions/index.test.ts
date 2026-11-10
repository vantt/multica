import { describe, expect, it } from "vitest";
import { decisionAnswerComment } from "./index";

describe("decisionAnswerComment", () => {
  const base = { question: "Ship it?", asked_by_type: "member", asked_by_id: "u-1" };

  it("is the question and the option when a person asked", () => {
    expect(decisionAnswerComment(base, "Yes", "")).toBe("**Ship it?** → Yes");
  });

  it("keeps the note", () => {
    expect(decisionAnswerComment(base, "No", "next week")).toBe("**Ship it?** → No\n\nnext week");
  });

  it("mentions the agent that asked, so it wakes", () => {
    const d = { ...base, asked_by_type: "agent", asked_by_id: "a-1" };
    expect(decisionAnswerComment(d, "Yes", "", "Phương (PM)")).toBe(
      "**Ship it?** → Yes\n\n[@Phương (PM)](mention://agent/a-1)",
    );
  });

  it("escapes brackets in the agent name", () => {
    const d = { ...base, asked_by_type: "agent", asked_by_id: "a-1" };
    expect(decisionAnswerComment(d, "Yes", "", "Bot [x]")).toBe("**Ship it?** → Yes\n\n[@Bot \\[x\\]](mention://agent/a-1)");
  });
});
