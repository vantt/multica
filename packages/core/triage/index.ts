import { api } from "../api";

/**
 * vantt triage-suggest (patch 0021, design D12): the server asks a fast decision
 * model (Cloudflare Clef) which project and Type a new issue belongs to, from the
 * workspace's own projects, Types and recent issues, and answers only what it is
 * at least 60% sure of. References: Linear Triage Intelligence (suggestions in
 * the create form), OpenMausBot's decision model (fail open, show the source).
 */
export interface TriageSuggestion {
  project?: { id: string; name: string; probability: number };
  type?: { property_id: string; option_id: string; label: string; probability: number };
  provider?: string;
}

export interface TriageInput {
  title: string;
  projectId?: string;
  hasType: boolean;
}

export const triageKeys = {
  suggest: (wsId: string, input: TriageInput) =>
    ["triage-suggest", wsId, input.title, input.projectId ?? "", input.hasType] as const,
};

// Plain options object so tests that mock react-query with only useQuery work.
export function triageSuggestOptions(wsId: string, input: TriageInput) {
  return {
    queryKey: triageKeys.suggest(wsId, input),
    queryFn: () => api.triageSuggest({ title: input.title, project_id: input.projectId, has_type: input.hasType }),
    staleTime: 5 * 60_000,
    retry: false,
  };
}

/** A title long enough to classify: at least 3 words or 12 characters. */
export function worthSuggesting(title: string): boolean {
  const t = title.trim();
  return t.length >= 12 || t.split(/\s+/).length >= 3;
}

export const percent = (p: number) => `${Math.round(p * 100)}%`;
