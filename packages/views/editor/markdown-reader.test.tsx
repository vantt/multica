import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../test/i18n";
import { MarkdownReader, markdownOutline } from "./markdown-reader";

afterEach(cleanup);

describe("markdownOutline", () => {
  it("lists headings outside code fences", () => {
    const text = "# Spec\n\n## Goals\n```\n# not a heading\n```\n### Notes *draft*\n#### too deep";
    expect(markdownOutline(text)).toEqual([
      { level: 1, text: "Spec" },
      { level: 2, text: "Goals" },
      { level: 3, text: "Notes draft" },
    ]);
  });
});

describe("MarkdownReader", () => {
  it("shows an outline for three headings and jumps to one", () => {
    const scroll = vi.fn();
    Element.prototype.scrollIntoView = scroll;
    renderWithI18n(
      <MarkdownReader text={"# A\n## B\n## C"} onBackdropClick={vi.fn()}>
        <h1>A</h1>
        <h2>B</h2>
        <h2>C</h2>
      </MarkdownReader>,
    );
    expect(screen.getByRole("navigation", { name: "On this page" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "C" }));
    expect(scroll).toHaveBeenCalled();
    expect(screen.getByRole("progressbar", { name: "Reading progress" })).toBeTruthy();
  });

  it("has no outline for a short note", () => {
    renderWithI18n(
      <MarkdownReader text={"# Only"} onBackdropClick={vi.fn()}>
        <h1>Only</h1>
      </MarkdownReader>,
    );
    expect(screen.queryByRole("navigation")).toBeNull();
  });
});
