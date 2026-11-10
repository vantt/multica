import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
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

export function workflowRulesOptions(wsId: string, projectId: string) {
  return queryOptions({
    queryKey: workflowRuleKeys.project(wsId, projectId),
    queryFn: () => api.listProjectWorkflowRules(projectId),
    select: (data) => data.rules,
    staleTime: 60_000,
  });
}

export function useSetWorkflowRules(projectId: string) {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (rules: WorkflowRule[]) => api.setProjectWorkflowRules(projectId, rules),
    onSuccess: () => qc.invalidateQueries({ queryKey: workflowRuleKeys.project(wsId, projectId) }),
  });
}
