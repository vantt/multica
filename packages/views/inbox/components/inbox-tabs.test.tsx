import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { effectiveTab, InboxTabs, itemsForTab, NeedsYouChip, needsYou } from "./inbox-tabs";

afterEach(cleanup);

const items = [
  { id: "a", severity: "action_required" as const, type: "decision_requested" as const },
  { id: "b", severity: "info" as const, type: "new_comment" as const },
  { id: "c", severity: "attention" as const, type: "autopilot_paused" as const },
];

describe("inbox tabs", () => {
  it("splits on the server's severity", () => {
    expect(items.filter(needsYou).map((i) => i.id)).toEqual(["a", "c"]);
    expect(itemsForTab(items, "needs").map((i) => i.id)).toEqual(["a", "c"]);
    expect(itemsForTab(items, "fyi").map((i) => i.id)).toEqual(["b"]);
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
  it("labels why a row needs you, and nothing for info rows", () => {
    renderWithI18n(<NeedsYouChip item={items[0]!} />);
    expect(screen.getByText("Decision needed")).toBeTruthy();
    const { container } = renderWithI18n(<NeedsYouChip item={items[1]!} />);
    expect(container.textContent).toBe("");
  });
});
