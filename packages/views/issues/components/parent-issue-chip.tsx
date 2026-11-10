"use client";

import { useQuery } from "@tanstack/react-query";
import { CornerDownRight } from "lucide-react";
import { useWorkspaceId } from "@multica/core/hooks";
import { issueDetailOptions } from "@multica/core/issues/queries";
import { cn } from "@multica/ui/lib/utils";
import { useT } from "../../i18n";

/**
 * Which parent a sub-issue belongs to (vantt parent-child, design D9), shown on
 * board cards and list rows next to the identifier. Reference: Jira's epic
 * link on cards; Linear's parent label on sub-issues. Parents already show their
 * sub-issue progress ring.
 */
export function ParentIssueChip({ parentId, className }: { parentId: string | null | undefined; className?: string }) {
  const { t } = useT("issues");
  const wsId = useWorkspaceId();
  const { data: parent } = useQuery({ ...issueDetailOptions(wsId, parentId ?? ""), enabled: Boolean(parentId) });
  if (!parentId || !parent) return null;
  return (
    <span
      title={t(($) => $.parent_chip.title, { identifier: parent.identifier, title: parent.title })}
      className={cn(
        "inline-flex min-w-0 max-w-[180px] shrink items-center gap-1 rounded-full bg-muted/60 px-1.5 py-0.5 text-micro text-muted-foreground",
        className,
      )}
    >
      <CornerDownRight aria-hidden className="size-3 shrink-0" />
      <span className="shrink-0">{parent.identifier}</span>
      <span className="truncate">{parent.title}</span>
    </span>
  );
}
