import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";

/**
 * vantt decisions (patch decisions-ui, F5): a question on an issue with options
 * for a person to pick, one recommended. Reference: Jira approvals.
 */
export interface DecisionOption {
  id: string;
  label: string;
  description: string;
  recommended: boolean;
}

export interface IssueDecision {
  id: string;
  issue_id: string;
  question: string;
  options: DecisionOption[];
  status: "open" | "answered" | "cancelled";
  due_date: string | null;
  asked_by_type: string;
  asked_by_id: string | null;
  answer_option_id: string | null;
  answer_note: string;
  answered_by: string | null;
  answered_at: string | null;
  created_at: string;
}

export const decisionKeys = {
  issue: (wsId: string, issueId: string) => ["decisions", wsId, issueId] as const,
};

// Plain options object so tests that mock react-query with only useQuery work.
export function issueDecisionsOptions(wsId: string, issueId: string) {
  return {
    queryKey: decisionKeys.issue(wsId, issueId),
    queryFn: () => api.listIssueDecisions(issueId),
    select: (data: { decisions: IssueDecision[] }) => data.decisions,
  };
}

/** Records the answer, then posts it as a comment so the issue's agent wakes up. */
export function useAnswerDecision(issueId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: async ({ decision, optionId, note }: { decision: IssueDecision; optionId: string; note: string }) => {
      const answered = await api.answerIssueDecision(issueId, decision.id, optionId, note);
      const label = decision.options.find((o) => o.id === optionId)?.label ?? optionId;
      await api.createComment(issueId, `**${decision.question}** → ${label}${note ? `\n\n${note}` : ""}`);
      return answered;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: decisionKeys.issue(wsId, issueId) }),
  });
}
