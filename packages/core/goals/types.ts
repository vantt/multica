/**
 * vantt goals (patch goals-ui, F2): results with a number and a deadline that
 * work moves. Health is computed by the server. Reference: Atlassian Goals.
 */
export type GoalStatus = "planned" | "active" | "achieved" | "missed" | "cancelled";
export type GoalHealth =
  | "on_track"
  | "at_risk"
  | "off_track"
  | "no_target_date"
  | "planned"
  | "achieved"
  | "missed"
  | "cancelled";

export interface GoalLink {
  target_type: "project" | "issue";
  target_id: string;
}

export interface Goal {
  id: string;
  workspace_id: string;
  parent_id: string | null;
  title: string;
  description: string;
  unit: string;
  direction: "increase" | "decrease";
  start_value: number;
  target_value: number;
  current_value: number;
  due_date: string | null;
  status: GoalStatus;
  measured_by: "manual" | "script";
  owner_type: string | null;
  owner_id: string | null;
  position: number;
  current_updated_at: string;
  created_at: string;
  updated_at: string;
  links: GoalLink[];
  progress: number;
  expected: number | null;
  health: GoalHealth;
  update_due: boolean;
  needed_per_week: number | null;
  actual_per_week: number | null;
}

export interface GoalHistoryEntry {
  id: string;
  value: number;
  note: string;
  actor_type: string;
  actor_id: string | null;
  created_at: string;
}

export interface GoalDetail {
  goal: Goal;
  history: GoalHistoryEntry[];
}

export interface GoalWriteRequest {
  parent_id?: string | null;
  title?: string;
  description?: string;
  unit?: string;
  direction?: "increase" | "decrease";
  start_value?: number;
  target_value?: number;
  due_date?: string | null;
  status?: GoalStatus;
  measured_by?: "manual" | "script";
}
