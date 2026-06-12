"use client";

import { shortenId } from "@/lib/shorten-id";
import { Cloud } from "@unkey/icons";
import Link from "next/link";

/**
 * Fourth crumb for the "crumb" deployment nav variant: appends the deployment
 * id to the top-bar trail (workspace › project › app › deployment).
 */
export function DeploymentCrumb({
  href,
  deploymentId,
}: {
  href: string;
  deploymentId: string;
}) {
  return (
    <Link
      href={href}
      aria-label={deploymentId}
      className="flex min-w-0 items-center gap-1.5 px-1 py-1 text-[13px] font-medium text-accent-12"
    >
      <Cloud className="size-3.5 text-accent-11" iconSize="sm-regular" />
      <span className="truncate font-mono max-w-[120px] md:max-w-[180px]">
        {shortenId(deploymentId)}
      </span>
    </Link>
  );
}
