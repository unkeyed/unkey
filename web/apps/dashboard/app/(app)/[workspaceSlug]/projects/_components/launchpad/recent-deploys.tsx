"use client";

import {
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import { cn } from "@/lib/utils";
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

/** GitHub serves an avatar for any handle at /<handle>.png, so a row still has
 * a face when the deployment row carried no avatar url. */
function avatarSrc(deploy: RecentDeployRow): string | null {
  if (deploy.authorAvatarUrl) {
    return deploy.authorAvatarUrl;
  }
  return deploy.authorHandle ? `https://github.com/${deploy.authorHandle}.png?size=40` : null;
}

function Avatar({ deploy }: { deploy: RecentDeployRow }) {
  const src = avatarSrc(deploy);
  if (src) {
    return (
      // biome-ignore lint/performance/noImgElement: avatars come from arbitrary git hosts
      <img
        src={src}
        alt=""
        className="size-5 shrink-0 rounded-full bg-grayA-3"
      />
    );
  }
  return (
    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-grayA-3 text-[9px] font-medium uppercase text-gray-11">
      {deploy.appName.slice(0, 2)}
    </span>
  );
}

/** One line: who, what branch, which app, when. Status rides the dot. */
export function DeployRow({ deploy, compact }: { deploy: RecentDeployRow; compact?: boolean }) {
  return (
    <Link
      href={deploy.href}
      className={cn(
        "group flex items-center gap-2 rounded-md px-2 text-[13px] transition-colors hover:bg-grayA-2",
        compact ? "h-9" : "h-10",
      )}
      title={deploy.commitMessage ?? undefined}
    >
      <Avatar deploy={deploy} />
      <span
        className={cn("size-1.5 shrink-0 rounded-full", statusTone(deploy.status))}
        aria-label={DEPLOYMENT_STATUS_LABELS[deploy.status]}
      />
      <span className="min-w-0 flex-1 truncate text-accent-12">
        {firstOf(deploy.branch, deploy.commitMessage, deploy.appName)}
      </span>
      <span className="shrink-0 truncate text-xs text-gray-9">{deploy.appName}</span>
      <TimestampInfo value={deploy.createdAt} className="shrink-0 text-[11px] text-gray-9" />
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
