"use client";

import type { InboxItem } from "@multica/core/types";
import { useT } from "../../i18n";
import { useTypeLabels } from "./inbox-detail-label";

/**
 * vantt inbox-split (patch 0020, design D11): the active inbox in two tabs.
 * "Needs you" holds the items the server already marks above info
 * (action_required: assigned to you, a decision to answer, a run that failed;
 * attention: an autopilot problem); "For your info" holds the rest. Archive is
 * still "Done". Reference: OpenMausBot's Needs you / For your info / Done.
 */
export type InboxTab = "needs" | "fyi";

export function needsYou(item: Pick<InboxItem, "severity">): boolean {
  return item.severity === "action_required" || item.severity === "attention";
}

/** The tab shown: the chosen one, else "Needs you" while it holds anything. */
export function effectiveTab(chosen: InboxTab | null, needsCount: number): InboxTab {
  return chosen ?? (needsCount > 0 ? "needs" : "fyi");
}

export function itemsForTab<T extends Pick<InboxItem, "severity">>(items: readonly T[], tab: InboxTab): T[] {
  return items.filter((item) => needsYou(item) === (tab === "needs"));
}

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
            on && value === "needs" && count > 0 ? "rounded-full bg-destructive px-1.5 text-white" : "text-muted-foreground"
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

/** Why a "Needs you" row is there: its kind, as a small chip. */
export function NeedsYouChip({ item }: { item: Pick<InboxItem, "severity" | "type"> }) {
  const labels = useTypeLabels();
  if (!needsYou(item)) return null;
  const label = labels[item.type] ?? item.type.replace(/_/g, " ");
  const tone =
    item.type === "decision_requested"
      ? "bg-violet-500/10 text-violet-700 dark:text-violet-300"
      : item.type === "task_failed" || item.severity === "attention"
        ? "bg-orange-500/10 text-orange-700 dark:text-orange-300"
        : "bg-red-500/10 text-red-700 dark:text-red-300";
  return <span className={`mr-1.5 inline-flex shrink-0 rounded-full px-1.5 text-micro font-medium ${tone}`}>{label}</span>;
}
