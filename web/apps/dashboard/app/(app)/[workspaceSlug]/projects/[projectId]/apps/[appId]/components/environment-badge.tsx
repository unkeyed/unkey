"use client";

import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import { cn } from "@/lib/utils";
import { Cloud, Eye } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

type EnvironmentBadgeProps = {
  environment: Environment;
  isCurrent: boolean;
};

export function EnvironmentBadge({ environment, isCurrent }: EnvironmentBadgeProps) {
  const isProduction = environment.kind === ENVIRONMENT_KIND.production;
  const Icon = isProduction ? Cloud : Eye;
  const isLive = isProduction && isCurrent;

  const badge = (
    <span
      className={cn(
        "inline-flex h-5.5 shrink-0 items-center gap-1.5 rounded-md border px-2 text-xs leading-none",
        isLive ? "border-transparent bg-info-9 text-white" : "border-grayA-5 text-accent-12",
      )}
    >
      <Icon iconSize="sm-regular" className="shrink-0" />
      <span className="capitalize">{environment.slug}</span>
    </span>
  );

  if (!isLive) {
    return badge;
  }

  return (
    <InfoTooltip
      content="This deployment is receiving production traffic."
      variant="inverted"
      position={{ side: "top" }}
      triggerClassName="relative z-20 inline-flex items-center"
    >
      {badge}
    </InfoTooltip>
  );
}
