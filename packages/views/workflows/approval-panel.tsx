"use client";

import { useState } from "react";
import { Lock } from "lucide-react";
import { useAuthStore } from "@multica/core/auth";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCreateComment } from "@multica/core/issues/mutations";
import { useProjectWithWorkflow, workflowNextStatus, workflowStep } from "@multica/core/issue-workflows";
import type { Issue, IssueWorkflowStep, UpdateIssueRequest } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { useStatusLabel } from "../issues/utils/status-label";
import { useT } from "../i18n";
import { stepHandlerActor } from "./step-handler";

/** Whether the viewer is the person a workflow step hands the issue to. */
export function stepWaitsOnViewer(
  step: IssueWorkflowStep | undefined,
  project: Parameters<typeof stepHandlerActor>[1],
  issue: Pick<Issue, "creator_type" | "creator_id">,
  viewerId: string | null | undefined,
): boolean {
  if (!step || !viewerId) return false;
  if (step.handler.type === "creator") return issue.creator_type === "member" && issue.creator_id === viewerId;
  const actor = stepHandlerActor(step, project);
  return actor?.type === "member" && actor.id === viewerId;
}

/**
 * Approval panel (vantt owner-gates, design D5b): shown while the issue sits in
 * a workflow step a person handles and the viewer is that person. Approve moves
 * it to the step's next status; Send back records why and moves it to the
 * step's back status. Agents cannot move the issue out of such a step.
 * Reference: Jira approvals (Approve / Decline).
 */
export function ApprovalPanel({
  issue,
  onUpdateField,
}: {
  issue: Issue;
  onUpdateField: (updates: Partial<UpdateIssueRequest>) => void;
}) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const viewerId = useAuthStore((s) => s.user?.id);
  const labelOf = useStatusLabel(wsId);
  const { project, workflow } = useProjectWithWorkflow(wsId, issue.project_id);
  const createComment = useCreateComment(issue.id);
  const [sendingBack, setSendingBack] = useState(false);
  const [reason, setReason] = useState("");

  const step = workflowStep(workflow, issue.status);
  if (!workflow || !stepWaitsOnViewer(step, project, issue, viewerId)) return null;
  const next = workflowNextStatus(workflow, issue.status);
  const back = step?.back_status_key ?? "";
  if (!next && !back) return null;

  const sendBack = () => {
    const note = reason.trim();
    if (!note) return;
    createComment.mutate(
      { content: note },
      {
        onSuccess: () => {
          onUpdateField({ status: back });
          setSendingBack(false);
          setReason("");
        },
      },
    );
  };

  return (
    <section
      aria-label={t(($) => $.workflows.approval.title)}
      className="mb-4 flex flex-col gap-3 rounded-md border border-amber-300 bg-amber-50 p-4 text-body dark:border-amber-500/40 dark:bg-amber-500/10"
    >
      <div className="flex flex-wrap items-center gap-2">
        <Lock aria-hidden className="size-4 shrink-0 text-amber-700 dark:text-amber-400" />
        <span className="font-semibold text-amber-900 dark:text-amber-200">{t(($) => $.workflows.approval.title)}</span>
        <span className="text-caption text-amber-800 dark:text-amber-300">{labelOf(issue.status)}</span>
      </div>
      <p className="text-caption text-amber-900 dark:text-amber-200">{t(($) => $.workflows.approval.gate_note)}</p>

      {!sendingBack ? (
        <div className="flex flex-wrap items-center gap-2">
          {next && (
            <Button type="button" onClick={() => onUpdateField({ status: next })}>
              {t(($) => $.workflows.approval.approve)}
            </Button>
          )}
          {back && (
            <Button type="button" variant="outline" onClick={() => setSendingBack(true)}>
              {t(($) => $.workflows.approval.send_back)}
            </Button>
          )}
          {next && (
            <span className="text-caption text-muted-foreground">
              {t(($) => $.workflows.approval.moves_to, { status: labelOf(next) })}
            </span>
          )}
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          <label htmlFor={`send-back-${issue.id}`} className="font-medium">
            {t(($) => $.workflows.approval.reason_label)}
          </label>
          <Textarea
            id={`send-back-${issue.id}`}
            rows={3}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            className="bg-background"
          />
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              onClick={sendBack}
              disabled={!reason.trim() || createComment.isPending}
              aria-busy={createComment.isPending}
            >
              {t(($) => $.workflows.approval.send_back_to, { status: labelOf(back) })}
            </Button>
            <Button type="button" variant="outline" onClick={() => setSendingBack(false)}>
              {t(($) => $.workflows.approval.cancel)}
            </Button>
          </div>
        </div>
      )}
    </section>
  );
}
