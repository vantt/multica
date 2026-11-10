"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { MessageCircleQuestion } from "lucide-react";
import { useAuthStore } from "@multica/core/auth";
import { useWorkspaceId } from "@multica/core/hooks";
import { issueDecisionsOptions, useAnswerDecision, type IssueDecision } from "@multica/core/decisions";
import type { Issue } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../i18n";

/**
 * Open decisions on an issue (vantt decisions-ui, design D4): the question, the
 * options with the recommended one marked, and Answer. The answer is posted as
 * a comment, which wakes the issue's agent. Reference: Jira approvals.
 */
export function DecisionPanel({ issue }: { issue: Issue }) {
  const wsId = useWorkspaceId();
  const { data: decisions = [] } = useQuery(issueDecisionsOptions(wsId, issue.id));
  const open = decisions.filter((d) => d.status === "open");
  if (open.length === 0) return null;
  return (
    <>
      {open.map((d) => (
        <DecisionCard key={d.id} issueId={issue.id} decision={d} />
      ))}
    </>
  );
}

function DecisionCard({ issueId, decision }: { issueId: string; decision: IssueDecision }) {
  const { t } = useT("issues");
  const isMember = useAuthStore((s) => Boolean(s.user?.id));
  const answer = useAnswerDecision(issueId);
  const recommended = decision.options.find((o) => o.recommended)?.id ?? decision.options[0]?.id ?? "";
  const [picked, setPicked] = useState(recommended);
  const [note, setNote] = useState("");
  const pickedLabel = decision.options.find((o) => o.id === picked)?.label ?? "";

  return (
    <section
      aria-label={t(($) => $.workflows.decision.title)}
      className="mb-4 flex flex-col gap-3 rounded-md border border-amber-300 bg-amber-50 p-4 text-body dark:border-amber-500/40 dark:bg-amber-500/10"
    >
      <div className="flex flex-wrap items-center gap-2">
        <MessageCircleQuestion aria-hidden className="size-4 shrink-0 text-amber-700 dark:text-amber-400" />
        <span className="text-caption font-semibold uppercase tracking-wide text-amber-900 dark:text-amber-200">
          {t(($) => $.workflows.decision.title)}
        </span>
        {decision.due_date && (
          <span className="text-caption text-amber-800 dark:text-amber-300">{t(($) => $.workflows.decision.due, { date: decision.due_date })}</span>
        )}
      </div>
      <p className="font-medium">{decision.question}</p>
      <div role="radiogroup" aria-label={decision.question} className="flex flex-col gap-2">
        {decision.options.map((o) => {
          const selected = o.id === picked;
          return (
            <button
              key={o.id}
              type="button"
              role="radio"
              aria-checked={selected}
              onClick={() => setPicked(o.id)}
              className={cn(
                "flex items-start gap-3 rounded-md border bg-background px-3 py-2.5 text-left",
                selected ? "border-foreground ring-1 ring-foreground" : "border-border",
              )}
            >
              <span
                aria-hidden
                className={cn("mt-1 size-3.5 shrink-0 rounded-full border-2 border-foreground", selected ? "bg-foreground" : "bg-background")}
              />
              <span className="flex min-w-0 flex-col gap-0.5">
                <span className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{o.label}</span>
                  {o.recommended && (
                    <span className="rounded-full bg-foreground px-2 py-0.5 text-caption font-medium text-background">
                      {t(($) => $.workflows.decision.recommended)}
                    </span>
                  )}
                </span>
                {o.description && <span className="text-caption text-muted-foreground">{o.description}</span>}
              </span>
            </button>
          );
        })}
      </div>
      {isMember && (
        <div className="flex flex-wrap items-center gap-2">
          <Input
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={t(($) => $.workflows.decision.note_placeholder)}
            aria-label={t(($) => $.workflows.decision.note_placeholder)}
            className="min-w-48 flex-1 bg-background"
          />
          <Button
            type="button"
            disabled={!picked || answer.isPending}
            aria-busy={answer.isPending}
            onClick={() => answer.mutate({ decision, optionId: picked, note: note.trim() })}
          >
            {t(($) => $.workflows.decision.answer, { option: pickedLabel })}
          </Button>
        </div>
      )}
    </section>
  );
}
