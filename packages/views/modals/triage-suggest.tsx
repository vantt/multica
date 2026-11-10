"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sparkles, X } from "lucide-react";
import { percent, triageSuggestOptions, worthSuggesting } from "@multica/core/triage";
import { useWorkspaceId } from "@multica/core/hooks";
import { useDebouncedValue } from "../common/use-debounced-value";
import { useT } from "../i18n";

/**
 * vantt triage-suggest (patch 0021, design D12): a "Suggested" strip in the
 * manual create form. Once the title has a few words and project or Type is
 * still empty, the server's decision model may suggest them; each chip applies
 * one field, "Apply both" applies all, ✕ hides it for this issue. Nothing is
 * shown when the model is off or unsure, and nothing is ever changed without a
 * click. Reference: Linear's create-form suggestions.
 */
export function TriageSuggestBar({
  title,
  projectId,
  typePropertyValue,
  onApplyProject,
  onApplyType,
}: {
  title: string;
  projectId?: string;
  /** The current value of the suggested Type property, by property id. */
  typePropertyValue: (propertyId: string) => unknown;
  onApplyProject: (id: string) => void;
  onApplyType: (propertyId: string, optionId: string) => void;
}) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const [dismissed, setDismissed] = useState(false);
  const debounced = useDebouncedValue(title.trim(), 600);
  const enabled = !dismissed && worthSuggesting(debounced);
  const { data } = useQuery({
    ...triageSuggestOptions(wsId, { title: debounced, projectId, hasType: false }),
    enabled,
  });
  if (dismissed || !data) return null;
  const project = !projectId ? data.project : undefined;
  const type = data.type && typePropertyValue(data.type.property_id) == null ? data.type : undefined;
  if (!project && !type) return null;
  const source = (p: number) => t(($) => $.triage.source, { percent: percent(p) });
  const chip = "inline-flex items-center gap-1 rounded-full border border-violet-300 bg-background px-2 py-0.5 text-caption text-violet-900 hover:bg-violet-50 dark:border-violet-700 dark:text-violet-200 dark:hover:bg-violet-950";
  return (
    <div
      role="group"
      aria-label={t(($) => $.triage.suggested)}
      className="mx-4 mb-1 flex flex-wrap items-center gap-2 rounded-md border border-violet-200 bg-violet-50 px-2.5 py-1.5 text-caption text-violet-800 dark:border-violet-800 dark:bg-violet-950/40 dark:text-violet-200"
    >
      <span className="inline-flex items-center gap-1 font-medium">
        <Sparkles className="size-3.5" aria-hidden />
        {t(($) => $.triage.suggested)}
      </span>
      {project && (
        <button type="button" className={chip} title={source(project.probability)} onClick={() => onApplyProject(project.id)}>
          <span>📁 {project.name}</span>
          <span className="text-micro text-violet-600 dark:text-violet-400">{percent(project.probability)}</span>
        </button>
      )}
      {type && (
        <button type="button" className={chip} title={source(type.probability)} onClick={() => onApplyType(type.property_id, type.option_id)}>
          <span>{type.label}</span>
          <span className="text-micro text-violet-600 dark:text-violet-400">{percent(type.probability)}</span>
        </button>
      )}
      <span className="flex-1" />
      {project && type && (
        <button
          type="button"
          className="font-medium hover:underline"
          onClick={() => {
            onApplyProject(project.id);
            onApplyType(type.property_id, type.option_id);
          }}
        >
          {t(($) => $.triage.apply_both)}
        </button>
      )}
      <button type="button" aria-label={t(($) => $.triage.dismiss)} className="rounded p-0.5 hover:bg-violet-100 dark:hover:bg-violet-900" onClick={() => setDismissed(true)}>
        <X className="size-3.5" aria-hidden />
      </button>
    </div>
  );
}
