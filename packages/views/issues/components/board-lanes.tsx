"use client";

import { createContext, useContext } from "react";
import { useQuery } from "@tanstack/react-query";
import { Lock } from "lucide-react";
import { gateStatuses } from "@multica/core/board-gates";
import { useIssueWorkflows, useProjectWorkflow, workflowAllowsStatus } from "@multica/core/issue-workflows";
import { useIssueStatuses } from "@multica/core/issue-statuses/hooks";
import { useWorkspaceId } from "@multica/core/hooks";
import { workflowRulesOptions } from "@multica/core/workflow-rules";
import type { Issue, IssueStatus, IssueStatusCategory, IssueWorkflow } from "@multica/core/types";
import { StatusIcon } from "./status-icon";
import { useT } from "../../i18n";
import type { IssueGroupPageState } from "../surface/use-issue-group-branches";
import type { BoardColumnGroup } from "./board-column";

/**
 * vantt board-lanes (designs D3, D3d "Automatic"): a project board whose Type
 * rules use several workflows shows four columns, each holding several statuses
 * — To do, In progress, Awaiting approval (every step a person approves), Done —
 * and each card names its real status. A board of one workflow keeps one column
 * per step. Reference: Jira board column mapping (several statuses per column).
 */
export type Lane = "todo" | "in_progress" | "awaiting_approval" | "done";
export const LANES: readonly Lane[] = ["todo", "in_progress", "awaiting_approval", "done"];
export const laneGroupId = (lane: Lane) => `lane:${lane}`;

export function laneOf(category: IssueStatusCategory, isGate: boolean): Lane {
  if (isGate) return "awaiting_approval";
  if (category === "unstarted") return "todo";
  if (category === "started") return "in_progress";
  return "done";
}

/** One group per lane that has a visible status, statuses kept in column order. */
export function buildLaneGroups(
  visibleStatuses: readonly IssueStatus[],
  categoryOf: (status: string) => IssueStatusCategory,
  gates: readonly string[],
  titleOf: (lane: Lane) => string,
): BoardColumnGroup[] {
  const byLane = new Map<Lane, IssueStatus[]>();
  for (const status of visibleStatuses) {
    const lane = laneOf(categoryOf(status), gates.includes(status));
    byLane.set(lane, [...(byLane.get(lane) ?? []), status]);
  }
  return LANES.filter((lane) => byLane.has(lane)).map((lane) => ({
    id: laneGroupId(lane),
    title: titleOf(lane),
    lane,
    statuses: byLane.get(lane)!,
    createData: lane === "awaiting_approval" ? undefined : { status: byLane.get(lane)![0]! },
  }));
}

/**
 * The status a card dropped on a lane takes: its own status when it is already
 * in the lane, else the lane's first status its workflow allows. Null when its
 * workflow has none of them.
 */
export function resolveLaneGroup(
  group: BoardColumnGroup | undefined,
  issue: Pick<Issue, "status"> | undefined,
  workflow: IssueWorkflow | null | undefined,
): BoardColumnGroup | undefined | null {
  if (!group?.statuses) return group;
  if (issue && group.statuses.includes(issue.status)) return { ...group, status: issue.status };
  const status = group.statuses.find((s) => !workflow || workflowAllowsStatus(workflow, s));
  return status ? { ...group, status } : null;
}

/** A lane's paging state: the merged pages of its statuses. */
export function lanePages(
  groups: readonly BoardColumnGroup[],
  statusPagination: Partial<Record<string, IssueGroupPageState>> | undefined,
): Record<string, IssueGroupPageState> {
  const out: Record<string, IssueGroupPageState> = {};
  if (!statusPagination) return out;
  for (const group of groups) {
    const pages = (group.statuses ?? []).flatMap((s) => (statusPagination[s] ? [statusPagination[s]!] : []));
    if (!pages.length) continue;
    out[group.id] = {
      total: pages.reduce((sum, p) => sum + p.total, 0),
      loaded: pages.reduce((sum, p) => sum + p.loaded, 0),
      hasMore: pages.some((p) => p.hasMore),
      isLoading: pages.some((p) => p.isLoading),
      isFetching: pages.some((p) => p.isFetching),
      isError: pages.some((p) => p.isError),
      loadMore: () => pages.forEach((p) => p.hasMore && p.loadMore()),
      retry: () => pages.forEach((p) => p.isError && p.retry()),
    };
  }
  return out;
}

export interface BoardLanes {
  gates: string[];
  /** The workflow an issue follows on this board: its Type rule's, else the project's. */
  workflowOf: (issue: Pick<Issue, "properties"> | null | undefined) => IssueWorkflow;
}

/** Lanes when this project's board uses several workflows (Type rules); null otherwise. */
export function useBoardLanes(wsId: string, projectId: string | null | undefined): BoardLanes | null {
  const { workflows } = useIssueWorkflows(wsId);
  const projectWorkflow = useProjectWorkflow(wsId, projectId);
  const { data: rules } = useQuery({ ...workflowRulesOptions(wsId, projectId ?? ""), enabled: Boolean(projectId) });
  if (!projectId || !projectWorkflow || !rules?.length) return null;
  const others = workflows.filter((w) => w.id !== projectWorkflow.id && rules.some((r) => r.workflow_id === w.id));
  if (!others.length) return null;
  const workflowOf = (issue: Pick<Issue, "properties"> | null | undefined) => {
    const rule = rules.find((r) => issue?.properties?.[r.property_id] === r.option_id);
    return (rule && workflows.find((w) => w.id === rule.workflow_id)) || projectWorkflow;
  };
  return { gates: gateStatuses([projectWorkflow, ...others]), workflowOf };
}

/** True inside a board whose columns are lanes: cards then name their status. */
export const BoardLaneContext = createContext(false);

/** A card's status on a lane board; custom statuses already show their own chip. */
export function LaneStatusChip({ status, isCustom }: { status: IssueStatus; isCustom: boolean }) {
  const lanes = useContext(BoardLaneContext);
  const catalog = useIssueStatuses(useWorkspaceId());
  const entry = catalog.entryOf(status);
  if (!lanes || isCustom || !entry) return null;
  return (
    <span className="inline-flex max-w-[160px] items-center gap-1 rounded-full bg-muted/60 px-1.5 py-0.5 text-micro text-muted-foreground">
      <StatusIcon status={status} category={catalog.categoryOf(status)} color={entry.color} icon={entry.icon} className="size-3" />
      <span className="truncate">{entry.name}</span>
    </span>
  );
}

export function LaneHeading({ group, count }: { group: BoardColumnGroup; count: number }) {
  const { t } = useT("issues");
  const gate = group.lane === "awaiting_approval";
  return (
    <span className="flex min-w-0 items-center gap-1.5" title={(group.statuses ?? []).join(", ")}>
      {gate && <Lock aria-label={t(($) => $.board_lanes.gate_hint)} className="size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />}
      <span className={gate ? "truncate font-medium text-amber-800 dark:text-amber-300" : "truncate font-medium"}>{group.title}</span>
      <span className="text-muted-foreground tabular-nums">{count}</span>
    </span>
  );
}
