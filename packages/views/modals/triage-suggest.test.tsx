import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../test/i18n";
import { TriageSuggestBar } from "./triage-suggest";

let suggestion: unknown = {};
vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "ws-1" }));
vi.mock("../common/use-debounced-value", () => ({ useDebouncedValue: (v: unknown) => v }));
vi.mock("@tanstack/react-query", () => ({ useQuery: (o: { enabled?: boolean }) => ({ data: o.enabled ? suggestion : undefined }) }));

afterEach(cleanup);

const both = {
  project: { id: "p1", name: "momentry", probability: 0.95 },
  type: { property_id: "type", option_id: "bug", label: "Tech · Bug", probability: 0.89 },
};

describe("TriageSuggestBar", () => {
  it("offers the sure fields and applies them on click", () => {
    suggestion = both;
    const onApplyProject = vi.fn();
    const onApplyType = vi.fn();
    renderWithI18n(<TriageSuggestBar title="Momentry crashes on long sessions" typePropertyValue={() => undefined} onApplyProject={onApplyProject} onApplyType={onApplyType} />);
    expect(screen.getByText("95%")).toBeTruthy();
    fireEvent.click(screen.getByText("Apply both"));
    expect(onApplyProject).toHaveBeenCalledWith("p1");
    expect(onApplyType).toHaveBeenCalledWith("type", "bug");
  });

  it("never offers a field already set, and hides on dismiss", () => {
    suggestion = both;
    renderWithI18n(<TriageSuggestBar title="Momentry crashes on long sessions" projectId="p9" typePropertyValue={() => undefined} onApplyProject={vi.fn()} onApplyType={vi.fn()} />);
    expect(screen.queryByText(/momentry/)).toBeNull();
    expect(screen.getByText("Tech · Bug")).toBeTruthy();
    fireEvent.click(screen.getByLabelText("Hide suggestion"));
    expect(screen.queryByText("Tech · Bug")).toBeNull();
  });

  it("shows nothing for a short title or an unsure model", () => {
    suggestion = both;
    const { container } = renderWithI18n(<TriageSuggestBar title="fix" typePropertyValue={() => undefined} onApplyProject={vi.fn()} onApplyType={vi.fn()} />);
    expect(container.textContent).toBe("");
    suggestion = {};
    const second = renderWithI18n(<TriageSuggestBar title="Momentry crashes on long sessions" typePropertyValue={() => undefined} onApplyProject={vi.fn()} onApplyType={vi.fn()} />);
    expect(second.container.textContent).toBe("");
  });
});
