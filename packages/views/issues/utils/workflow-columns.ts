import { useQuery } from "@tanstack/react-query";
import { defaultWorkflow, useIssueWorkflows, useProjectWorkflow } from "@multica/core/issue-workflows";
import { mergeRuleWorkflows, workflowRulesOptions } from "@multica/core/workflow-rules";
import type { IssueStatus, IssueWorkflow } from "@multica/core/types";

/**
 * The status columns of a surface whose columns follow a workflow (MUL-7420):
 * the workflow's steps, in workflow order, narrowed to `visible` (hidden-column
 * preferences and status filters). Without a workflow the catalog-ordered
 * `visible` list passes through unchanged. `visible === null` returns every
 * step.
 */
export function workflowColumns(
  visible: readonly IssueStatus[] | null,
  workflow: IssueWorkflow | null,
): IssueStatus[] {
  if (!workflow) return visible ? [...visible] : [];
  const keys = workflow.steps.map((s) => s.status_key as IssueStatus);
  if (!visible) return keys;
  const allowed = new Set(visible);
  return keys.filter((key) => allowed.has(key));
}

/**
 * The workflow a surface's columns follow (MUL-7420). A project surface shows
 * its project's columns. Elsewhere, columns follow Default.
 * A status outside the workflow is never a column; a status filter
 * that picks keys explicitly shows exactly those — on a project too, which is
 * how an issue left in a status its workflow no longer lists is found.
 */
export function useColumnWorkflow(
  wsId: string,
  projectId: string | null | undefined,
  hasStatusFilter: boolean,
): { workflow: IssueWorkflow | null } {
  const { workflows } = useIssueWorkflows(wsId);
  const projectWorkflow = useProjectWorkflow(wsId, projectId);
  // vantt board-columns: a project whose rules send some Types to other
  // workflows shows their steps too, so every issue has a column.
  const { data: rules } = useQuery({ ...workflowRulesOptions(wsId, projectId ?? ""), enabled: Boolean(projectId) });
  if (hasStatusFilter) return { workflow: null };
  if (projectId) return { workflow: mergeRuleWorkflows(projectWorkflow, rules ?? [], workflows) };
  return { workflow: defaultWorkflow(workflows) };
}
