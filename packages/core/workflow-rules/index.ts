import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";

/**
 * vantt workflow-by-type (patch workflow-by-type-ui, F3): in a project, issues
 * whose select property has an option follow another workflow. Reference: Jira
 * workflow schemes.
 */
export interface WorkflowRule {
  property_id: string;
  option_id: string;
  workflow_id: string;
}

export const workflowRuleKeys = {
  project: (wsId: string, projectId: string) => ["workflow-rules", wsId, projectId] as const,
};

// A plain options object (not queryOptions) so surfaces whose tests mock
// react-query with only useQuery can still read rules.
export function workflowRulesOptions(wsId: string, projectId: string) {
  return {
    queryKey: workflowRuleKeys.project(wsId, projectId),
    queryFn: () => api.listProjectWorkflowRules(projectId),
    select: (data: { rules: WorkflowRule[] }) => data.rules,
    staleTime: 60_000,
  };
}

/**
 * A project's columns when rules send some Types to other workflows (vantt
 * board-columns, design D3d "Automatic"): the project workflow's steps, then
 * each rule workflow's middle steps it does not have yet, before the closing
 * steps (blocked, done, cancelled). Without rules: the project workflow.
 */
export function mergeRuleWorkflows<W extends { steps: { status_key: string }[] }>(
  project: W | null,
  rules: readonly WorkflowRule[],
  workflows: readonly (W & { id: string })[],
  tail: readonly string[] = ["blocked", "done", "cancelled"],
): W | null {
  if (!project || rules.length === 0) return project;
  const steps = [...project.steps];
  const has = new Set(steps.map((s) => s.status_key));
  const tailAt = () => {
    const i = steps.findIndex((s) => tail.includes(s.status_key));
    return i < 0 ? steps.length : i;
  };
  for (const rule of rules) {
    const wf = workflows.find((w) => w.id === rule.workflow_id);
    for (const step of wf?.steps ?? []) {
      if (has.has(step.status_key) || tail.includes(step.status_key)) continue;
      steps.splice(tailAt(), 0, step);
      has.add(step.status_key);
    }
  }
  return { ...project, steps };
}

/**
 * The properties a project's rules read (e.g. Type): the issue's workflow depends
 * on them, so the create form and the issue page show them without "Add property"
 * (vantt workflow-by-type; Jira puts the work type first).
 */
export function ruleProperties(rules: readonly WorkflowRule[] | undefined): Set<string> {
  return new Set((rules ?? []).map((r) => r.property_id));
}

export function useSetWorkflowRules(projectId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (rules: WorkflowRule[]) => api.setProjectWorkflowRules(projectId, rules),
    onSuccess: () => qc.invalidateQueries({ queryKey: workflowRuleKeys.project(wsId, projectId) }),
  });
}
