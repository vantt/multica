"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../i18n";

/**
 * Reading view for a markdown document (vantt artifact-reader, design D10): a
 * reading-progress bar and, for documents with at least three headings, a table
 * of contents beside the text on wide screens. Reference: GitHub's markdown
 * outline, Notion and Confluence page outlines.
 */

export interface OutlineItem {
  level: number;
  text: string;
}

/** Headings (#, ##, ###) of a markdown text, outside fenced code blocks. */
export function markdownOutline(text: string): OutlineItem[] {
  const out: OutlineItem[] = [];
  let fence: string | null = null;
  for (const line of text.split("\n")) {
    const f = line.match(/^\s*(```|~~~)/);
    if (f) {
      fence = fence === f[1] ? null : fence ?? f[1]!;
      continue;
    }
    if (fence) continue;
    const m = line.match(/^(#{1,3})\s+(.+?)\s*#*\s*$/);
    if (m) out.push({ level: m[1]!.length, text: m[2]!.replace(/[*_`]/g, "") });
  }
  return out;
}

export function MarkdownReader({
  text,
  children,
  onBackdropClick,
}: {
  text: string;
  children: ReactNode;
  onBackdropClick: (e: React.MouseEvent) => void;
}) {
  const { t } = useT("editor");
  const scrollRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const outline = useMemo(() => markdownOutline(text), [text]);
  const showOutline = outline.length >= 3;
  const [progress, setProgress] = useState(0);
  const [active, setActive] = useState(0);

  const headings = useCallback(
    () => Array.from(bodyRef.current?.querySelectorAll("h1, h2, h3") ?? []) as HTMLElement[],
    [],
  );

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onScroll = () => {
      const max = el.scrollHeight - el.clientHeight;
      setProgress(max > 0 ? el.scrollTop / max : 1);
      const top = el.getBoundingClientRect().top + 80;
      let current = 0;
      headings().forEach((h, i) => {
        if (h.getBoundingClientRect().top <= top) current = i;
      });
      setActive(current);
    };
    onScroll();
    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, [headings]);

  const jump = (index: number) => {
    headings()[index]?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  return (
    <div ref={scrollRef} className="relative h-full overflow-y-auto" onClick={onBackdropClick}>
      <div
        role="progressbar"
        aria-label={t(($) => $.reader.progress)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(progress * 100)}
        className="sticky top-0 z-10 h-0.5 w-full bg-transparent"
      >
        <div className="h-0.5 bg-brand transition-[width]" style={{ width: `${Math.round(progress * 100)}%` }} />
      </div>
      <div className="mx-auto flex w-full max-w-6xl items-start gap-6">
        {showOutline && (
          <nav
            aria-label={t(($) => $.reader.outline)}
            onClick={(e) => e.stopPropagation()}
            className="sticky top-4 mt-8 hidden max-h-[calc(100vh-4rem)] w-56 shrink-0 overflow-y-auto rounded-lg bg-background/90 p-3 text-caption xl:block"
          >
            <p className="mb-2 font-semibold text-foreground">{t(($) => $.reader.outline)}</p>
            <ol className="flex flex-col gap-0.5">
              {outline.map((item, i) => (
                <li key={`${i}-${item.text}`}>
                  <button
                    type="button"
                    onClick={() => jump(i)}
                    aria-current={i === active ? "location" : undefined}
                    className={cn(
                      "w-full truncate rounded px-1.5 py-1 text-left hover:bg-accent",
                      item.level === 2 && "pl-3.5",
                      item.level === 3 && "pl-6",
                      i === active ? "font-medium text-foreground" : "text-muted-foreground",
                    )}
                  >
                    {item.text}
                  </button>
                </li>
              ))}
            </ol>
          </nav>
        )}
        <div
          ref={bodyRef}
          className="mx-auto min-h-full w-full max-w-3xl rounded-t-lg bg-background text-foreground leading-relaxed"
        >
          {children}
        </div>
      </div>
    </div>
  );
}
