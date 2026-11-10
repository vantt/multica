"use client";

import { useQuery } from "@tanstack/react-query";
import { ExternalLink, GitPullRequest } from "lucide-react";
import { issuePullRequestsOptions } from "@multica/core/github/queries";
import { issueTimelineOptions } from "@multica/core/issues/queries";
import type { Issue } from "@multica/core/types";
import { useT } from "../i18n";

const PR_URL = /https:\/\/github\.com\/([\w.-]+)\/([\w.-]+)\/pull\/(\d+)/g;

export interface ApprovalPr {
  url: string;
  repo: string;
  number: number;
  title?: string;
  state?: string;
}

/**
 * Pull requests of an issue for the approval panel (vantt approval-pr, design D5b):
 * the PRs linked to the issue, then GitHub PR links written in its description and
 * comments (agents post "Approve PR <link>"), newest mention first, one per PR.
 */
export function collectPrs(
  linked: readonly { html_url: string; repo_owner: string; repo_name: string; number: number; title: string; state: string }[],
  texts: readonly string[],
): ApprovalPr[] {
  const out: ApprovalPr[] = [];
  const seen = new Set<string>();
  const add = (pr: ApprovalPr) => {
    const key = `${pr.repo}#${pr.number}`.toLowerCase();
    if (seen.has(key)) return;
    seen.add(key);
    out.push(pr);
  };
  for (const pr of linked) add({ url: pr.html_url, repo: `${pr.repo_owner}/${pr.repo_name}`, number: pr.number, title: pr.title, state: pr.state });
  for (const text of [...texts].reverse()) {
    for (const m of text.matchAll(PR_URL)) add({ url: m[0], repo: `${m[1]}/${m[2]}`, number: Number(m[3]) });
  }
  return out;
}

export function ApprovalPullRequests({ issue }: { issue: Pick<Issue, "id" | "description"> }) {
  const { t } = useT("issues");
  const { data: linked } = useQuery({ ...issuePullRequestsOptions(issue.id), retry: false });
  const { data: timeline } = useQuery(issueTimelineOptions(issue.id));
  const texts = [issue.description ?? "", ...(timeline ?? []).filter((e) => e.type === "comment").map((e) => e.content ?? "")];
  const prs = collectPrs(linked?.pull_requests ?? [], texts).slice(0, 3);
  if (!prs.length) return null;
  return (
    <ul aria-label={t(($) => $.workflows.approval.prs_label)} className="flex flex-col gap-1.5">
      {prs.map((pr) => (
        <li key={pr.url} className="flex flex-wrap items-center gap-2 rounded-md border border-amber-200 bg-background px-3 py-2 dark:border-amber-500/30">
          <GitPullRequest aria-hidden className="size-4 shrink-0 text-muted-foreground" />
          <a href={pr.url} target="_blank" rel="noreferrer" className="min-w-0 truncate font-medium hover:underline">
            {pr.repo}#{pr.number}{pr.title ? ` · ${pr.title}` : ""}
          </a>
          {pr.state && <span className="text-caption text-muted-foreground">{pr.state}</span>}
          <a
            href={`${pr.url}/files`}
            target="_blank"
            rel="noreferrer"
            className="ml-auto inline-flex items-center gap-1 rounded-md border border-input bg-background px-2.5 py-1 text-caption font-medium hover:bg-accent"
          >
            {t(($) => $.workflows.approval.approve_on_github)}
            <ExternalLink aria-hidden className="size-3" />
          </a>
        </li>
      ))}
      <li className="text-caption text-amber-900 dark:text-amber-200">{t(($) => $.workflows.approval.pr_then_approve)}</li>
    </ul>
  );
}
