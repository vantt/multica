import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";
import type { Goal } from "./types";

export const goalKeys = {
  all: (wsId: string) => ["goals", wsId] as const,
  list: (wsId: string) => [...goalKeys.all(wsId), "list"] as const,
  detail: (wsId: string, id: string) => [...goalKeys.all(wsId), "detail", id] as const,
};

export function goalListOptions(wsId: string) {
  return queryOptions({
    queryKey: goalKeys.list(wsId),
    queryFn: () => api.listGoals(),
    select: (data) => data.goals,
    staleTime: 60_000,
  });
}

export function goalDetailOptions(wsId: string, id: string) {
  return queryOptions({
    queryKey: goalKeys.detail(wsId, id),
    queryFn: () => api.getGoal(id),
  });
}

/** Goals as a tree: roots first, each followed by its children, in position order. */
export function goalTree(goals: readonly Goal[]): { goal: Goal; depth: number }[] {
  const byParent = new Map<string | null, Goal[]>();
  const ids = new Set(goals.map((g) => g.id));
  for (const g of goals) {
    const key = g.parent_id && ids.has(g.parent_id) ? g.parent_id : null;
    byParent.set(key, [...(byParent.get(key) ?? []), g]);
  }
  const out: { goal: Goal; depth: number }[] = [];
  const walk = (parent: string | null, depth: number) => {
    for (const g of byParent.get(parent) ?? []) {
      out.push({ goal: g, depth });
      walk(g.id, depth + 1);
    }
  };
  walk(null, 0);
  return out;
}

/** The goals that a project or an issue (or its parent) moves. */
export function goalsForTarget(goals: readonly Goal[], targetType: "project" | "issue", ids: (string | null | undefined)[]): Goal[] {
  const want = new Set(ids.filter(Boolean) as string[]);
  return goals.filter((g) => g.links.some((l) => l.target_type === targetType && want.has(l.target_id)));
}
