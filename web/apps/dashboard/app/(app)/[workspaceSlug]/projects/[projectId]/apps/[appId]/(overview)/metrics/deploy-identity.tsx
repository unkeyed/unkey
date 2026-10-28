"use client";

import { imageRefDisplay } from "@/lib/docker-image-ref";
import type { DeploySource } from "@/lib/metrics/app-metrics";
import { IconCodeCommitOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { cn } from "cn";
import type { ReactNode } from "react";
import { Avatar } from "../../components/git-avatar";

export function sourceLabel(source: DeploySource | null, deploymentId: string): string {
  if (!source) {
    return deploymentId;
  }
  return match(source)
    .with({ kind: "git" }, ({ sha }) => sha.slice(0, 7))
    .with({ kind: "image" }, ({ ref }) => imageRefDisplay(ref))
    .exhaustive();
}

export function commitSubject(source: DeploySource | null): string | null {
  return source?.kind === "git" ? source.message?.split("\n")[0].trim() || null : null;
}

export function DeployIdentity({
  source,
  deploymentId,
  aside,
  className,
  children,
}: {
  source: DeploySource | null;
  deploymentId: string;
  aside?: ReactNode;
  className?: string;
  children?: ReactNode;
}) {
  const subject = commitSubject(source);
  const git = source?.kind === "git" ? source : null;
  return (
    <div className={cn("flex flex-col", className)}>
      <div className="flex items-center justify-between gap-2">
        <span className={cn("truncate text-gray-12", !subject && "font-mono")}>
          {subject ?? sourceLabel(source, deploymentId)}
        </span>
        {aside}
      </div>
      {git && (subject || git.authorHandle) && (
        <span className="flex min-w-0 items-center gap-1.5 text-gray-10">
          {subject && (
            <>
              <IconCodeCommitOutline18 className="size-3 shrink-0 text-gray-9" />
              <span className="font-mono">{sourceLabel(git, deploymentId)}</span>
            </>
          )}
          {subject && git.authorHandle && (
            <span className="text-gray-8" aria-hidden="true">
              ·
            </span>
          )}
          {git.authorHandle && (
            <>
              <Avatar src={git.avatarUrl} alt={git.authorHandle} className="size-4" />
              <span className="truncate">{git.authorHandle}</span>
            </>
          )}
        </span>
      )}
      {children}
    </div>
  );
}
