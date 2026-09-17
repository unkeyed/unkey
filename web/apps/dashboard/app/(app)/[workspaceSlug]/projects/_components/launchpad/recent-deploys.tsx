"use client";

import {
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import { cn } from "@/lib/utils";
import { IconCodeBranchOutline18, IconCubeOutline18 } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import Link from "next/link";
import { Section } from "./parts";
import type { Chrome, RecentDeployRow } from "./types";

function firstOf(...candidates: (string | null)[]): string {
  return candidates.find((candidate) => candidate && candidate.trim().length > 0) ?? "Deployment";
}

function statusTone(status: DeploymentStatus): string {
  if (status === "ready") {
    return "bg-success-9";
  }
  if (status === "failed" || status === "cancelled") {
    return "bg-error-9";
  }
  return isDeploymentInFlight(status) ? "bg-warning-9" : "bg-gray-8";
}

function Avatar({ deploy }: { deploy: RecentDeployRow }) {
  if (deploy.authorAvatarUrl) {
    return (
      // biome-ignore lint/performance/noImgElement: avatars come from arbitrary git hosts
      <img
        src={deploy.authorAvatarUrl}
        alt=""
        className="size-5 shrink-0 rounded-full ring-1 ring-grayA-4"
      />
    );
  }
  return (
    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-grayA-3 text-[9px] font-medium uppercase text-gray-11">
      {(deploy.authorHandle ?? deploy.appName).slice(0, 2)}
    </span>
  );
}

function Pill({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex min-w-0 max-w-full items-center gap-1 rounded border border-grayA-4 px-1 py-px text-[11px] text-gray-11">
      {children}
    </span>
  );
}

/** Vercel's Recent Previews shape: who and what on top, where it went below. */
export function DeployRow({ deploy, compact }: { deploy: RecentDeployRow; compact?: boolean }) {
  return (
    <Link
      href={deploy.href}
      className={cn(
        "group flex items-start gap-2.5 rounded-md px-2 transition-colors hover:bg-grayA-2",
        compact ? "py-1.5" : "py-2",
      )}
    >
      <Avatar deploy={deploy} />
      <span className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="flex min-w-0 items-center gap-2">
          <span className="min-w-0 flex-1 truncate text-[13px] text-accent-12">
            {firstOf(deploy.commitMessage, deploy.branch, deploy.appName)}
          </span>
          <TimestampInfo value={deploy.createdAt} className="shrink-0 text-[11px] text-gray-9" />
        </span>
        <span className="flex min-w-0 flex-wrap items-center gap-1">
          <Pill>
            <span className={cn("size-1.5 shrink-0 rounded-full", statusTone(deploy.status))} />
            <span className="truncate">{DEPLOYMENT_STATUS_LABELS[deploy.status]}</span>
          </Pill>
          {deploy.branch && (
            <Pill>
              <IconCodeBranchOutline18 className="size-2.5 shrink-0 text-gray-9" />
              <span className="truncate">
                {deploy.prNumber ? `#${deploy.prNumber}` : deploy.branch}
              </span>
            </Pill>
          )}
          <Pill>
            <IconCubeOutline18 className="size-2.5 shrink-0 text-gray-9" />
            <span className="truncate">{deploy.appName}</span>
          </Pill>
        </span>
      </span>
    </Link>
  );
}

export function DeployList({
  title,
  chrome,
  deploys,
  limit = 4,
  empty,
  compact,
}: {
  title: string;
  chrome: Chrome;
  deploys: RecentDeployRow[];
  limit?: number;
  empty: string;
  compact?: boolean;
}) {
  return (
    <Section title={title} chrome={chrome} count={deploys.length || undefined}>
      {deploys.length === 0 ? (
        <p className="px-2 py-1.5 text-xs text-gray-9">{empty}</p>
      ) : (
        deploys.slice(0, limit).map((deploy) => (
          <DeployRow key={deploy.id} deploy={deploy} compact={compact} />
        ))
      )}
    </Section>
  );
}
