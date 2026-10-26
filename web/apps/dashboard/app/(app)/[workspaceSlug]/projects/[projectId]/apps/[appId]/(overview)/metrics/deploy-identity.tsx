"use client";

import { imageRefDisplay } from "@/lib/docker-image-ref";
import type { DeploySource } from "@/lib/metrics/app-metrics";
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
  return source?.kind === "git" ? (source.message?.split("\n")[0] ?? null) : null;
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
  return (
    <div className={cn("flex flex-col", className)}>
      <div className="flex items-center justify-between gap-2">
        <span className="truncate font-mono text-gray-12">{sourceLabel(source, deploymentId)}</span>
        {aside}
      </div>
      {subject && <span className="truncate text-gray-11">{subject}</span>}
      {source?.kind === "git" && source.authorHandle && (
        <span className="flex items-center gap-1.5 text-gray-10">
          <Avatar src={source.avatarUrl} alt={source.authorHandle} className="size-4" />
          {source.authorHandle}
        </span>
      )}
      {children}
    </div>
  );
}
