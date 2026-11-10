import { api } from "../api";

/**
 * vantt board-gates (design D3): days in column on board cards, and the steps a
 * person approves, which a board of several workflows folds into one
 * "Awaiting approval" column. Reference: Jira board ("days in column", column
 * mapping several statuses).
 */
export interface StatusSince {
  status: string;
  since: string;
}

export const statusSinceKeys = {
  scope: (wsId: string, projectId: string | null) => ["status-since", wsId, projectId ?? "all"] as const,
};

// Plain options object so tests that mock react-query with only useQuery work.
export function statusSinceOptions(wsId: string, projectId: string | null) {
  return {
    queryKey: statusSinceKeys.scope(wsId, projectId),
    queryFn: () => api.listIssueStatusSince(projectId),
    select: (data: { issues: Record<string, StatusSince> }) => data.issues,
    staleTime: 60_000,
  };
}

/** Whole days an issue has sat in `status`; null when unknown or the issue moved since. */
export function daysInColumn(entry: StatusSince | undefined, status: string, now: number = Date.now()): number | null {
  if (!entry || entry.status !== status) return null;
  const at = Date.parse(entry.since);
  if (Number.isNaN(at)) return null;
  return Math.max(0, Math.floor((now - at) / 86_400_000));
}

interface GateStep {
  status_key: string;
  handler: { type: string };
  next_status_key?: string | null;
}

/**
 * Approval steps: a person handles them (member, project lead or creator) and
 * they lead somewhere (`next_status_key`). Blocked-style steps without a next
 * status are not approvals.
 */
export function gateStatuses(workflows: readonly { steps: readonly GateStep[] }[]): string[] {
  const out: string[] = [];
  for (const wf of workflows) {
    for (const step of wf.steps) {
      const person = ["member", "project_lead", "creator"].includes(step.handler?.type);
      if (person && step.next_status_key && !out.includes(step.status_key)) out.push(step.status_key);
    }
  }
  return out;
}
