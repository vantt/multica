import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { InboxItem } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";
import { effectiveTab, InboxTabs, itemsForTab, NeedsYouChip, NeedsYouContext, needsYouIndex } from "./inbox-tabs";

afterEach(cleanup);

const n = (o: Partial<InboxItem>) => ({ archived: false, created_at: "2026-11-10T00:00:00Z", issue_status: "todo", ...o }) as InboxItem;
const raw = [
  n({ id: "1", issue_id: "A", severity: "action_required", type: "issue_assigned", created_at: "2026-11-09T00:00:00Z" }),
  n({ id: "2", issue_id: "A", severity: "info", type: "new_comment" }), // A's newest is a comment
  n({ id: "3", issue_id: "B", severity: "info", type: "status_changed" }),
  n({ id: "4", issue_id: "C", severity: "action_required", type: "issue_assigned", issue_status: "done" }),
  n({ id: "5", issue_id: "D", severity: "action_required", type: "decision_requested", archived: true }),
];
const rows = [raw[1]!, raw[2]!, raw[3]!]; // grouped rows: newest per issue, unarchived
const closed = (i: InboxItem) => i.issue_status === "done";

describe("inbox tabs", () => {
  it("puts a row in Needs you when any of its notifications asks, even if the newest is a comment", () => {
    const index = needsYouIndex(raw, closed);
    expect([...index.keys()]).toEqual(["A"]);
    expect(itemsForTab(rows, index, "needs").map((r) => r.issue_id)).toEqual(["A"]);
    expect(itemsForTab(rows, index, "fyi").map((r) => r.issue_id)).toEqual(["B", "C"]);
  });
  it("opens on Needs you while it holds anything, else on For your info", () => {
    expect(effectiveTab(null, 2)).toBe("needs");
    expect(effectiveTab(null, 0)).toBe("fyi");
    expect(effectiveTab("fyi", 2)).toBe("fyi");
  });
  it("renders both tabs with counts and switches", () => {
    const onChange = vi.fn();
    renderWithI18n(<InboxTabs tab="needs" needsCount={2} fyiCount={5} onChange={onChange} />);
    expect(screen.getByRole("tab", { name: /Needs you/ }).getAttribute("aria-selected")).toBe("true");
    fireEvent.click(screen.getByRole("tab", { name: /For your info/ }));
    expect(onChange).toHaveBeenCalledWith("fyi");
  });
  it("labels a row with the notification that asks", () => {
    const index = needsYouIndex(raw, closed);
    renderWithI18n(<NeedsYouContext.Provider value={index}><NeedsYouChip item={rows[0]!} /></NeedsYouContext.Provider>);
    expect(screen.getByText("Assigned")).toBeTruthy();
    const { container } = renderWithI18n(<NeedsYouContext.Provider value={index}><NeedsYouChip item={rows[1]!} /></NeedsYouContext.Provider>);
    expect(container.textContent).toBe("");
  });
});
