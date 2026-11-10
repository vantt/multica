import { z } from "zod";
import type { Goal, GoalDetail } from "./types";

const HEALTHS = ["on_track", "at_risk", "off_track", "no_target_date", "planned", "achieved", "missed", "cancelled"] as const;
const STATUSES = ["planned", "active", "achieved", "missed", "cancelled"] as const;

const num = z.number().catch(0);
const optNum = z.number().nullable().catch(null);
const str = z.string().catch("");
const optStr = z.string().nullable().catch(null);

export const GoalSchema = z.object({
  id: z.string(),
  workspace_id: str,
  parent_id: optStr,
  title: str,
  description: str,
  unit: str,
  direction: z.enum(["increase", "decrease"]).catch("increase"),
  start_value: num,
  target_value: num,
  current_value: num,
  due_date: optStr,
  status: z.enum(STATUSES).catch("active"),
  measured_by: z.enum(["manual", "script"]).catch("manual"),
  owner_type: optStr,
  owner_id: optStr,
  position: num,
  current_updated_at: str,
  created_at: str,
  updated_at: str,
  links: z
    .array(z.object({ target_type: z.enum(["project", "issue"]), target_id: z.string() }))
    .catch([]),
  progress: num,
  expected: optNum,
  // Unknown health values from a newer server read as on-track-neutral.
  health: z.enum(HEALTHS).catch("no_target_date"),
  update_due: z.boolean().catch(false),
  needed_per_week: optNum,
  actual_per_week: optNum,
});

export const ListGoalsSchema = z.object({ goals: z.array(GoalSchema).catch([]) });

export const GoalDetailSchema = z.object({
  goal: GoalSchema,
  history: z
    .array(
      z.object({
        id: z.string(),
        value: num,
        note: str,
        actor_type: str,
        actor_id: optStr,
        created_at: str,
      }),
    )
    .catch([]),
});

export const EMPTY_GOALS: { goals: Goal[] } = { goals: [] };
export const EMPTY_GOAL_DETAIL: GoalDetail | null = null;
