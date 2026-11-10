"use client";

import { useQuery } from "@tanstack/react-query";
import { useWorkspaceId } from "@multica/core/hooks";
import { daysInColumn, statusSinceOptions } from "@multica/core/board-gates";
import { useIssueStatuses } from "@multica/core/issue-statuses/hooks";
import type { Issue } from "@multica/core/types";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";
import { useStatusLabel } from "../utils/status-label";

const CATEGORY_ORDER = ["done", "started", "unstarted", "backlog", "closed", "cancelled"];

/**
 * Child issues by status (vantt parent-child, design D9): one bar segment per
 * status, coloured like the status, done first. Reference: Jira's "Child issues"
 * progress bar.
 */
export function ChildStatusBar({ issues }: { issues: readonly Pick<Issue, "status">[] }) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const catalog = useIssueStatuses(wsId);
  const labelOf = useStatusLabel(wsId);
  if (issues.length === 0) return null;
  const counts = new Map<string, number>();
  for (const c of issues) counts.set(c.status, (counts.get(c.status) ?? 0) + 1);
  const rank = (s: string) => {
    const i = CATEGORY_ORDER.indexOf(catalog.categoryOf(s));
    return i < 0 ? CATEGORY_ORDER.length : i;
  };
  const parts = [...counts].sort((a, b) => rank(a[0]) - rank(b[0]) || b[1] - a[1]);
  const summary = parts.map(([s, n]) => `${n} ${labelOf(s)}`).join(", ");
  return (
    <span
      role="img"
      aria-label={t(($) => $.child_panel.status_bar, { summary })}
      title={summary}
      className="flex h-1.5 w-24 shrink-0 overflow-hidden rounded-full bg-muted sm:w-32"
    >
      {parts.map(([status, n]) => (
        <span
          key={status}
          style={{ width: `${(n / issues.length) * 100}%`, backgroundColor: catalog.colorOf(status) ?? undefined }}
          className="h-full"
        />
      ))}
    </span>
  );
}

/**
 * Days a child has sat in its status (vantt parent-child, design D9); shown from
 * one day on, red from five. Done and cancelled children show nothing.
 */
export function DaysInStatus({ issue, className }: { issue: Pick<Issue, "id" | "status" | "project_id">; className?: string }) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const labelOf = useStatusLabel(wsId);
  const { data } = useQuery(statusSinceOptions(wsId, issue.project_id ?? null));
  const days = daysInColumn(data?.[issue.id], issue.status);
  if (days === null || days < 1) return null;
  return (
    <span
      title={t(($) => $.child_panel.days_title, { count: days, status: labelOf(issue.status) })}
      className={cn(
        "shrink-0 text-caption tabular-nums",
        days >= 5 ? "font-medium text-destructive" : "text-muted-foreground",
        className,
      )}
    >
      {t(($) => $.child_panel.days, { count: days })}
    </span>
  );
}
