"use client";

import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useWorkspaceId } from "@multica/core/hooks";
import { useIssueWorkflows } from "@multica/core/issue-workflows";
import { propertyListOptions } from "@multica/core/properties/queries";
import type { Project } from "@multica/core/types";
import { useSetWorkflowRules, workflowRulesOptions, type WorkflowRule } from "@multica/core/workflow-rules";
import { Button } from "@multica/ui/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@multica/ui/components/ui/popover";
import { useT } from "../i18n";
import { useWorkflowPermissions } from "./permissions";

/**
 * Workflows by Type (vantt workflow-by-type-ui, design D6): in this project an
 * issue whose select property (Type) has an option follows another workflow;
 * anything else follows the project's workflow. Reference: Jira workflow schemes.
 */
export function WorkflowRulesControl({ project }: { project: Project }) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const { workflows } = useIssueWorkflows(wsId);
  const { data: properties = [] } = useQuery(propertyListOptions(wsId));
  const { data: rules = [] } = useQuery(workflowRulesOptions(wsId, project.id));
  const { canManageProject, isAdmin } = useWorkflowPermissions(project);
  const save = useSetWorkflowRules(project.id);
  const selects = useMemo(() => properties.filter((p) => p.type === "select" && !p.archived), [properties]);
  const [open, setOpen] = useState(false);
  const [propertyId, setPropertyId] = useState("");
  const [draft, setDraft] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!open) return;
    const pid = rules[0]?.property_id ?? selects.find((p) => p.name.toLowerCase() === "type")?.id ?? selects[0]?.id ?? "";
    setPropertyId(pid);
    setDraft(Object.fromEntries(rules.filter((r) => r.property_id === pid).map((r) => [r.option_id, r.workflow_id])));
  }, [open, rules, selects]);

  const property = selects.find((p) => p.id === propertyId);
  const nameOfWorkflow = (id: string) => workflows.find((w) => w.id === id)?.name ?? "?";
  const optionName = (r: WorkflowRule) =>
    properties.find((p) => p.id === r.property_id)?.config.options?.find((o) => o.id === r.option_id)?.name ?? "?";

  const summary =
    rules.length === 0
      ? t(($) => $.workflows.rules.none)
      : rules.map((r) => `${optionName(r)} → ${nameOfWorkflow(r.workflow_id)}`).join(", ");

  if (!(canManageProject || isAdmin)) {
    return <span className="truncate text-caption text-muted-foreground">{summary}</span>;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <button type="button" className="-ml-1 min-w-0 truncate rounded-md px-1 py-0.5 text-left text-caption hover:bg-accent">
            {summary}
          </button>
        }
      />
      <PopoverContent align="start" className="w-96 p-3">
        <p className="mb-2 text-caption text-muted-foreground">{t(($) => $.workflows.rules.help)}</p>
        <label className="mb-3 flex items-center gap-2 text-caption">
          <span className="text-muted-foreground">{t(($) => $.workflows.rules.choose_by)}</span>
          <select
            className="h-8 flex-1 rounded-md border bg-background px-2 text-body"
            value={propertyId}
            onChange={(e) => {
              setPropertyId(e.target.value);
              setDraft({});
            }}
          >
            {selects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </label>
        {property && (
          <ul className="mb-3 flex max-h-72 flex-col gap-1.5 overflow-y-auto">
            {(property.config.options ?? []).map((o) => (
              <li key={o.id} className="flex items-center gap-2 text-caption">
                <span className="w-36 truncate">{o.name}</span>
                <span aria-hidden className="text-muted-foreground">→</span>
                <select
                  aria-label={o.name}
                  className="h-7 flex-1 rounded-md border bg-background px-1.5 text-caption"
                  value={draft[o.id] ?? ""}
                  onChange={(e) => setDraft((d) => ({ ...d, [o.id]: e.target.value }))}
                >
                  <option value="">{t(($) => $.workflows.rules.project_default)}</option>
                  {workflows.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </li>
            ))}
          </ul>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => setOpen(false)}>
            {t(($) => $.workflows.rules.cancel)}
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={save.isPending}
            aria-busy={save.isPending}
            onClick={() =>
              save.mutate(
                Object.entries(draft)
                  .filter(([, wf]) => wf)
                  .map(([option_id, workflow_id]) => ({ property_id: propertyId, option_id, workflow_id })),
                { onSuccess: () => setOpen(false) },
              )
            }
          >
            {t(($) => $.workflows.rules.save)}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
