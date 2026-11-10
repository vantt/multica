"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Target } from "lucide-react";
import { useWorkspaceId } from "@multica/core/hooks";
import { useWorkspacePaths } from "@multica/core/paths";
import {
  goalDetailOptions,
  goalListOptions,
  goalTree,
  useAddGoalValue,
  useCreateGoal,
  type Goal,
  type GoalHealth,
} from "@multica/core/goals";
import { issueDetailOptions } from "@multica/core/issues/queries";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { cn } from "@multica/ui/lib/utils";
import { PageHeader, PAGE_GUTTER, PAGE_RAIL } from "../layout/page-header";
import { AppLink } from "../navigation";
import { useT } from "../i18n";

/**
 * Goals (vantt patch goals-ui, design D1 and D2): results with a number and a
 * deadline, their health, and the work that moves them. Reference: Atlassian
 * Goals (On track / At risk / Off track), Linear initiatives.
 */

const HEALTH_CLASS: Record<GoalHealth, string> = {
  on_track: "bg-emerald-100 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  at_risk: "bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  off_track: "bg-red-100 text-red-800 dark:bg-red-500/15 dark:text-red-300",
  no_target_date: "bg-muted text-muted-foreground",
  planned: "bg-muted text-muted-foreground",
  achieved: "bg-emerald-100 text-emerald-800 dark:bg-emerald-500/15 dark:text-emerald-300",
  missed: "bg-red-100 text-red-800 dark:bg-red-500/15 dark:text-red-300",
  cancelled: "bg-muted text-muted-foreground",
};

const BAR_CLASS: Record<GoalHealth, string> = {
  on_track: "bg-emerald-600",
  at_risk: "bg-amber-500",
  off_track: "bg-red-600",
  no_target_date: "bg-foreground/60",
  planned: "bg-foreground/40",
  achieved: "bg-emerald-600",
  missed: "bg-red-600",
  cancelled: "bg-foreground/30",
};

function fmt(n: number, unit: string): string {
  const v = Math.abs(n) >= 100 ? Math.round(n).toLocaleString() : Number(n.toFixed(3)).toLocaleString();
  return unit ? `${v} ${unit}` : v;
}

function useHealthLabel() {
  const { t } = useT("projects");
  return (h: GoalHealth) => t(($) => $.goals.health[h]);
}

function HealthPill({ goal }: { goal: Goal }) {
  const label = useHealthLabel();
  const { t } = useT("projects");
  return (
    <span className={cn("inline-flex items-center rounded-full px-2 py-0.5 text-caption font-medium", HEALTH_CLASS[goal.health])}>
      {goal.update_due ? t(($) => $.goals.update_due) : label(goal.health)}
    </span>
  );
}

/** Where the goal lives in the repo, and whether UI changes still wait to be written there. */
function RepoSource({ goal, unsynced = 0 }: { goal: Goal; unsynced?: number }) {
  const { t } = useT("projects");
  const waiting = goal.pending || unsynced > 0 || !goal.source_ref;
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5 text-caption text-muted-foreground">
      {goal.source_ref ? <code className="rounded bg-muted px-1 py-0.5">{goal.source_ref}</code> : null}
      {waiting && (
        <span className="rounded-full bg-amber-100 px-2 py-0.5 font-medium text-amber-800 dark:bg-amber-500/15 dark:text-amber-300">
          {t(($) => $.goals.waiting_repo)}
        </span>
      )}
    </span>
  );
}

function ProgressBar({ goal }: { goal: Goal }) {
  return (
    <div className="h-1.5 overflow-hidden rounded-full bg-muted" aria-hidden>
      <div className={cn("h-1.5 rounded-full", BAR_CLASS[goal.health])} style={{ width: `${Math.round(goal.progress * 100)}%` }} />
    </div>
  );
}

function PaceText({ goal }: { goal: Goal }) {
  const { t } = useT("projects");
  if (goal.needed_per_week == null) return null;
  return (
    <span className="text-caption text-muted-foreground">
      {t(($) => $.goals.pace, {
        needed: fmt(goal.needed_per_week ?? 0, ""),
        actual: fmt(goal.actual_per_week ?? 0, ""),
      })}
    </span>
  );
}

export function GoalsPage({ goalId }: { goalId?: string }) {
  return goalId ? <GoalDetailView goalId={goalId} /> : <GoalListView />;
}

function GoalListView() {
  const { t } = useT("projects");
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { data: goals = [], isLoading } = useQuery(goalListOptions(wsId));
  const [creating, setCreating] = useState(false);
  const tree = useMemo(() => goalTree(goals), [goals]);
  const open = goals.filter((g) => g.status === "active" || g.status === "planned");
  const count = (pred: (g: Goal) => boolean) => open.filter(pred).length;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <Target aria-hidden className="size-4 text-muted-foreground" />
        <h1 className="text-body font-semibold">{t(($) => $.goals.title)}</h1>
        <div className="ml-auto">
          <Button type="button" size="sm" onClick={() => setCreating((v) => !v)}>
            {t(($) => $.goals.new_goal)}
          </Button>
        </div>
      </PageHeader>
      <div className={cn(PAGE_RAIL, PAGE_GUTTER, "flex flex-col gap-4 overflow-y-auto py-5")}>
        <p className="text-caption text-muted-foreground">{t(($) => $.goals.subtitle)}</p>
        <p className="text-caption text-muted-foreground">{t(($) => $.goals.repo_note)}</p>
        {creating && <NewGoalForm goals={goals} onDone={() => setCreating(false)} />}
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Tile label={t(($) => $.goals.health.off_track)} value={count((g) => g.health === "off_track")} tone="text-red-700 dark:text-red-400" />
          <Tile label={t(($) => $.goals.health.at_risk)} value={count((g) => g.health === "at_risk")} tone="text-amber-700 dark:text-amber-400" />
          <Tile label={t(($) => $.goals.health.on_track)} value={count((g) => g.health === "on_track")} tone="text-emerald-700 dark:text-emerald-400" />
          <Tile label={t(($) => $.goals.update_due)} value={count((g) => g.update_due)} tone="text-amber-700 dark:text-amber-400" />
        </div>
        {!isLoading && goals.length === 0 ? (
          <p className="rounded-md border border-dashed p-6 text-center text-body text-muted-foreground">{t(($) => $.goals.empty)}</p>
        ) : (
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full min-w-[760px] text-body">
              <thead>
                <tr className="border-b text-left text-caption text-muted-foreground">
                  <th className="px-3 py-2 font-medium">{t(($) => $.goals.col_goal)}</th>
                  <th className="px-3 py-2 font-medium">{t(($) => $.goals.col_progress)}</th>
                  <th className="px-3 py-2 font-medium">{t(($) => $.goals.col_health)}</th>
                  <th className="px-3 py-2 font-medium">{t(($) => $.goals.col_measured)}</th>
                  <th className="px-3 py-2 font-medium">{t(($) => $.goals.col_work)}</th>
                </tr>
              </thead>
              <tbody>
                {tree.map(({ goal, depth }) => (
                  <tr key={goal.id} className="border-b last:border-0 hover:bg-accent/50">
                    <td className="px-3 py-2.5" style={{ paddingLeft: `${12 + depth * 20}px` }}>
                      <AppLink href={paths.goalDetail(goal.id)} className="font-medium hover:underline">
                        {goal.title}
                      </AppLink>
                      {goal.due_date && (
                        <div className="text-caption text-muted-foreground">{t(($) => $.goals.due, { date: goal.due_date })}</div>
                      )}
                      <RepoSource goal={goal} />
                    </td>
                    <td className="w-56 px-3 py-2.5">
                      <div className="mb-1 flex justify-between text-caption">
                        <span className="font-semibold">{fmt(goal.current_value, goal.unit)}</span>
                        <span className="text-muted-foreground">{t(($) => $.goals.of, { target: fmt(goal.target_value, goal.unit) })}</span>
                      </div>
                      <ProgressBar goal={goal} />
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="flex flex-col items-start gap-1">
                        <HealthPill goal={goal} />
                        <PaceText goal={goal} />
                      </div>
                    </td>
                    <td className="px-3 py-2.5 text-caption">
                      {goal.measured_by === "script" ? t(($) => $.goals.measured_script) : t(($) => $.goals.measured_manual)}
                    </td>
                    <td className="px-3 py-2.5 text-caption text-muted-foreground">
                      {t(($) => $.goals.work_count, { count: goal.links.length })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function Tile({ label, value, tone }: { label: string; value: number; tone: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-md border bg-card px-3.5 py-3">
      <span className="text-caption text-muted-foreground">{label}</span>
      <span className={cn("text-xl font-semibold", tone)}>{value}</span>
    </div>
  );
}

function NewGoalForm({ goals, onDone }: { goals: Goal[]; onDone: () => void }) {
  const { t } = useT("projects");
  const create = useCreateGoal();
  const [title, setTitle] = useState("");
  const [target, setTarget] = useState("");
  const [unit, setUnit] = useState("");
  const [due, setDue] = useState("");
  const [parent, setParent] = useState("");
  const [direction, setDirection] = useState<"increase" | "decrease">("increase");
  const [start, setStart] = useState("0");
  const valid = title.trim() !== "" && target.trim() !== "" && !Number.isNaN(Number(target));

  return (
    <form
      className="flex flex-col gap-3 rounded-md border p-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!valid) return;
        create.mutate(
          {
            title: title.trim(),
            target_value: Number(target),
            start_value: Number(start) || 0,
            unit: unit.trim(),
            due_date: due || null,
            parent_id: parent || null,
            direction,
          },
          { onSuccess: onDone },
        );
      }}
    >
      <div className="grid gap-3 md:grid-cols-3">
        <label className="flex flex-col gap-1 text-caption md:col-span-3">
          {t(($) => $.goals.field_title)}
          <Input value={title} onChange={(e) => setTitle(e.target.value)} required />
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_start)}
          <Input inputMode="decimal" value={start} onChange={(e) => setStart(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_target)}
          <Input inputMode="decimal" value={target} onChange={(e) => setTarget(e.target.value)} required />
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_unit)}
          <Input value={unit} onChange={(e) => setUnit(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_due)}
          <Input type="date" value={due} onChange={(e) => setDue(e.target.value)} />
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_direction)}
          <select
            className="h-8 rounded-md border bg-background px-2 text-body"
            value={direction}
            onChange={(e) => setDirection(e.target.value as "increase" | "decrease")}
          >
            <option value="increase">{t(($) => $.goals.direction_increase)}</option>
            <option value="decrease">{t(($) => $.goals.direction_decrease)}</option>
          </select>
        </label>
        <label className="flex flex-col gap-1 text-caption">
          {t(($) => $.goals.field_parent)}
          <select className="h-8 rounded-md border bg-background px-2 text-body" value={parent} onChange={(e) => setParent(e.target.value)}>
            <option value="">{t(($) => $.goals.no_parent)}</option>
            {goals.map((g) => (
              <option key={g.id} value={g.id}>
                {g.title}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="flex gap-2">
        <Button type="submit" disabled={!valid || create.isPending} aria-busy={create.isPending}>
          {t(($) => $.goals.create)}
        </Button>
        <Button type="button" variant="outline" onClick={onDone}>
          {t(($) => $.goals.cancel)}
        </Button>
      </div>
    </form>
  );
}

function GoalChart({ goal, history }: { goal: Goal; history: { value: number; created_at: string }[] }) {
  const { t } = useT("projects");
  const startT = Date.parse(goal.created_at);
  const endT = goal.due_date ? Date.parse(goal.due_date) + 86_400_000 : Date.now();
  const span = Math.max(1, endT - startT);
  const lo = Math.min(goal.start_value, goal.target_value, ...history.map((h) => h.value));
  const hi = Math.max(goal.start_value, goal.target_value, ...history.map((h) => h.value));
  const range = hi - lo || 1;
  const x = (ts: number) => Math.max(0, Math.min(600, ((ts - startT) / span) * 600));
  const y = (v: number) => 200 - ((v - lo) / range) * 190 - 5;
  const points = [{ value: goal.start_value, created_at: goal.created_at }, ...history]
    .map((h) => `${x(Date.parse(h.created_at)).toFixed(1)},${y(h.value).toFixed(1)}`)
    .join(" ");
  return (
    <figure className="flex flex-col gap-1">
      <svg
        viewBox="0 0 600 200"
        preserveAspectRatio="none"
        className="h-48 w-full border-b border-l"
        role="img"
        aria-label={t(($) => $.goals.chart_label)}
      >
        <line x1="0" y1={y(goal.start_value)} x2="600" y2={y(goal.target_value)} className="stroke-muted-foreground" strokeWidth="1.5" strokeDasharray="6 6" />
        <polyline points={points} fill="none" className={cn("stroke-current", goal.health === "off_track" ? "text-red-600" : goal.health === "at_risk" ? "text-amber-500" : "text-emerald-600")} strokeWidth="2.5" />
      </svg>
      <figcaption className="flex justify-between text-caption text-muted-foreground">
        <span>{goal.created_at.slice(0, 10)}</span>
        <span>{t(($) => $.goals.chart_needed)}</span>
        <span>{goal.due_date ?? ""}</span>
      </figcaption>
    </figure>
  );
}

function LinkedIssue({ id }: { id: string }) {
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { data: issue } = useQuery(issueDetailOptions(wsId, id));
  if (!issue) return null;
  return (
    <li className="flex items-center gap-2 border-b py-2 text-body last:border-0">
      <span className="w-20 shrink-0 text-caption font-medium text-muted-foreground">{issue.identifier}</span>
      <AppLink href={paths.issueDetail(issue.id)} className="min-w-0 flex-1 truncate hover:underline">
        {issue.title}
      </AppLink>
      <span className="text-caption text-muted-foreground">{issue.status}</span>
    </li>
  );
}

function GoalDetailView({ goalId }: { goalId: string }) {
  const { t } = useT("projects");
  const wsId = useWorkspaceId();
  const paths = useWorkspacePaths();
  const { data } = useQuery(goalDetailOptions(wsId, goalId));
  const addValue = useAddGoalValue();
  const [value, setValue] = useState("");
  const [note, setNote] = useState("");
  if (!data) return null;
  const { goal, history } = data;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <AppLink href={paths.goals()} className="text-body text-muted-foreground hover:text-foreground">
          {t(($) => $.goals.title)}
        </AppLink>
        <span className="text-muted-foreground">/</span>
        <h1 className="truncate text-body font-semibold">{goal.title}</h1>
      </PageHeader>
      <div className={cn(PAGE_RAIL, PAGE_GUTTER, "flex max-w-5xl flex-col gap-5 overflow-y-auto py-5")}>
        <div className="flex flex-wrap items-center gap-3">
          <HealthPill goal={goal} />
          <RepoSource goal={goal} unsynced={history.filter((h) => !h.synced).length} />
          {goal.due_date && <span className="text-caption text-muted-foreground">{t(($) => $.goals.due, { date: goal.due_date })}</span>}
        </div>
        <div className="flex flex-wrap items-baseline gap-2">
          <span className="text-3xl font-bold">{fmt(goal.current_value, goal.unit)}</span>
          <span className="text-muted-foreground">{t(($) => $.goals.of, { target: fmt(goal.target_value, goal.unit) })}</span>
          <span className="ml-auto">
            <PaceText goal={goal} />
          </span>
        </div>
        <GoalChart goal={goal} history={history} />

        <form
          className="flex flex-wrap items-end gap-2 rounded-md border p-3"
          onSubmit={(e) => {
            e.preventDefault();
            const v = Number(value);
            if (value.trim() === "" || Number.isNaN(v)) return;
            addValue.mutate({ id: goal.id, value: v, note }, { onSuccess: () => { setValue(""); setNote(""); } });
          }}
        >
          <label className="flex flex-col gap-1 text-caption">
            {t(($) => $.goals.enter_number)}
            <Input inputMode="decimal" value={value} onChange={(e) => setValue(e.target.value)} className="w-32" />
          </label>
          <label className="flex min-w-48 flex-1 flex-col gap-1 text-caption">
            {t(($) => $.goals.note)}
            <Input value={note} onChange={(e) => setNote(e.target.value)} />
          </label>
          <Button type="submit" disabled={addValue.isPending} aria-busy={addValue.isPending}>
            {t(($) => $.goals.save_number)}
          </Button>
        </form>

        <section className="flex flex-col gap-1">
          <h2 className="text-body font-semibold">{t(($) => $.goals.work_title)}</h2>
          {goal.links.filter((l) => l.target_type === "issue").length === 0 ? (
            <p className="text-caption text-muted-foreground">{t(($) => $.goals.no_work)}</p>
          ) : (
            <ul>
              {goal.links
                .filter((l) => l.target_type === "issue")
                .map((l) => (
                  <LinkedIssue key={l.target_id} id={l.target_id} />
                ))}
            </ul>
          )}
        </section>

        <section className="flex flex-col gap-1">
          <h2 className="text-body font-semibold">{t(($) => $.goals.history_title)}</h2>
          <ul className="flex flex-col gap-1 text-caption text-muted-foreground">
            {[...history].reverse().map((h) => (
              <li key={h.id}>
                {h.created_at.slice(0, 16).replace("T", " ")} · {fmt(h.value, goal.unit)}
                {h.note ? ` · ${h.note}` : ""}
              </li>
            ))}
            <li>{t(($) => $.goals.created_on, { date: goal.created_at.slice(0, 10) })}</li>
          </ul>
        </section>
      </div>
    </div>
  );
}
