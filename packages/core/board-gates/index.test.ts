import { describe, expect, it } from "vitest";
import { daysInColumn, gateStatuses } from "./index";

describe("daysInColumn", () => {
  const now = Date.parse("2026-11-10T12:00:00Z");
  it("counts whole days in the current status", () => {
    expect(daysInColumn({ status: "in_review", since: "2026-11-05T11:00:00Z" }, "in_review", now)).toBe(5);
    expect(daysInColumn({ status: "in_review", since: "2026-11-10T08:00:00Z" }, "in_review", now)).toBe(0);
  });
  it("is unknown once the issue moved or without an entry", () => {
    expect(daysInColumn({ status: "todo", since: "2026-11-01T00:00:00Z" }, "in_review", now)).toBeNull();
    expect(daysInColumn(undefined, "todo", now)).toBeNull();
  });
});

describe("gateStatuses", () => {
  it("lists steps a person approves, once, skipping blocked-style steps", () => {
    const delivery = { steps: [
      { status_key: "todo", handler: { type: "agent" }, next_status_key: "build" },
      { status_key: "spec_approval", handler: { type: "member" }, next_status_key: "build" },
      { status_key: "in_review", handler: { type: "member" }, next_status_key: "release" },
      { status_key: "blocked", handler: { type: "member" }, next_status_key: null },
    ] };
    const content = { steps: [{ status_key: "in_review", handler: { type: "project_lead" }, next_status_key: "done" }] };
    expect(gateStatuses([delivery, content])).toEqual(["spec_approval", "in_review"]);
  });
});
