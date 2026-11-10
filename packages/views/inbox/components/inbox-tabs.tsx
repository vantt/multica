"use client";

import { createContext, useContext } from "react";
import type { InboxItem } from "@multica/core/types";
import { useT } from "../../i18n";
import { useTypeLabels } from "./inbox-detail-label";

/**
 * vantt inbox-split (patch 0020, design D11): the active inbox in two tabs.
 * A row (one issue, its notifications grouped) is in "Needs you" when any of
 * its unarchived notifications is marked above info by the server
 * (action_required: assigned to you, a decision to answer, a run that failed;
 * attention: an autopilot problem) and its issue is not closed; everything else
 * is "For your info". The row list shows only each issue's newest
 * notification, which is often a comment, so the grouped rows alone cannot
 * tell. Archive is still "Done". Reference: OpenMausBot's Needs you / For your
 * info / Done.
 */
export type InboxTab = "needs" | "fyi";

export const rowKey = (item: Pick<InboxItem, "issue_id" | "id">) => item.issue_id ?? item.id;

export function asksSomething(item: Pick<InboxItem, "severity">): boolean {
  return item.severity === "action_required" || item.severity === "attention";
}

/**
 * Row key → the newest notification that asks something of the person, over
 * every unarchived notification (not only each row's newest). Rows whose issue
 * is closed are left out: an old "assigned to you" on a finished issue asks
 * nothing any more.
 */
export function needsYouIndex(
  items: readonly InboxItem[],
  isClosed: (item: InboxItem) => boolean,
): Map<string, InboxItem> {
  const index = new Map<string, InboxItem>();
  for (const item of items) {
    if (item.archived || !asksSomething(item) || isClosed(item)) continue;
    const key = rowKey(item);
    const prev = index.get(key);
    if (!prev || prev.created_at < item.created_at) index.set(key, item);
  }
  return index;
}

/** The tab shown: the chosen one, else "Needs you" while it holds anything. */
export function effectiveTab(chosen: InboxTab | null, needsCount: number): InboxTab {
  return chosen ?? (needsCount > 0 ? "needs" : "fyi");
}

export function itemsForTab<T extends Pick<InboxItem, "issue_id" | "id">>(
  rows: readonly T[],
  index: ReadonlyMap<string, unknown>,
  tab: InboxTab,
): T[] {
  return rows.filter((row) => index.has(rowKey(row)) === (tab === "needs"));
}

export const EMPTY_NEEDS: ReadonlyMap<string, InboxItem> = new Map();

/** The index, for the chip on each row. */
export const NeedsYouContext = createContext<ReadonlyMap<string, InboxItem>>(EMPTY_NEEDS);

export function InboxTabs({
  tab,
  needsCount,
  fyiCount,
  onChange,
}: {
  tab: InboxTab;
  needsCount: number;
  fyiCount: number;
  onChange: (tab: InboxTab) => void;
}) {
  const { t } = useT("inbox");
  const button = (value: InboxTab, label: string, count: number) => {
    const on = tab === value;
    return (
      <button
        type="button"
        role="tab"
        aria-selected={on}
        onClick={() => onChange(value)}
        className={`flex items-center gap-1.5 rounded-md px-2 py-1 text-caption outline-none transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
          on ? "bg-accent font-medium text-foreground" : "text-muted-foreground hover:bg-accent/50"
        }`}
      >
        {label}
        <span
          className={`tabular-nums text-micro ${
            value === "needs" && count > 0 ? "rounded-full bg-destructive px-1.5 text-white" : "text-muted-foreground"
          }`}
        >
          {count}
        </span>
      </button>
    );
  };
  return (
    <div role="tablist" aria-label={t(($) => $.tabs.label)} className="flex shrink-0 items-center gap-1 border-b px-3 py-1.5">
      {button("needs", t(($) => $.tabs.needs_you), needsCount)}
      {button("fyi", t(($) => $.tabs.for_your_info), fyiCount)}
    </div>
  );
}

/** Why a "Needs you" row is there: the kind of the notification that asks. */
export function NeedsYouChip({ item }: { item: Pick<InboxItem, "issue_id" | "id"> }) {
  const labels = useTypeLabels();
  const asking = useContext(NeedsYouContext).get(rowKey(item));
  if (!asking) return null;
  const label = labels[asking.type] ?? asking.type.replace(/_/g, " ");
  const tone =
    asking.type === "decision_requested"
      ? "bg-violet-500/10 text-violet-700 dark:text-violet-300"
      : asking.type === "task_failed" || asking.severity === "attention"
        ? "bg-orange-500/10 text-orange-700 dark:text-orange-300"
        : "bg-red-500/10 text-red-700 dark:text-red-300";
  return <span className={`mr-1.5 inline-flex shrink-0 rounded-full px-1.5 text-micro font-medium ${tone}`}>{label}</span>;
}
