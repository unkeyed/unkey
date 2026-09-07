"use client";

import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import { cn } from "@/lib/utils";
import { ArrowDotAntiClockwise, CircleXMark, Cloud, Eye } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { format } from "date-fns";

type EnvironmentBadgeProps = {
  environment: Environment;
  isCurrent: boolean;
  isRolledBack?: boolean;
  isRolledBackFrom?: boolean;
  liveSince?: number | null;
};

export function EnvironmentBadge({
  environment,
  isCurrent,
  isRolledBack = false,
  isRolledBackFrom = false,
  liveSince = null,
}: EnvironmentBadgeProps) {
  const isProduction = environment.kind === ENVIRONMENT_KIND.production;
  const isLive = isProduction && isCurrent;
  const wasRolledBack = isProduction && isRolledBackFrom;
  const Icon = wasRolledBack
    ? CircleXMark
    : isLive && isRolledBack
      ? ArrowDotAntiClockwise
      : isProduction
        ? Cloud
        : Eye;

  const badge = (
    <span
      className={cn(
        "inline-flex h-5.5 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs leading-none",
        isLive
          ? "border-transparent bg-info-11 text-white dark:bg-info-9 dark:text-gray-1"
          : wasRolledBack
            ? "border-transparent bg-errorA-3 text-error-11"
            : "border-grayA-5 text-accent-12",
      )}
    >
      <Icon iconSize="sm-regular" className="shrink-0" />
      <span className="capitalize">{environment.slug}</span>
    </span>
  );

  if (!isProduction) {
    return badge;
  }

  return (
    <InfoTooltip
      content={
        <ProductionTooltip
          isLive={isLive}
          isRolledBack={isRolledBack}
          wasRolledBack={wasRolledBack}
          liveSince={liveSince}
        />
      }
      variant="inverted"
      position={{ side: "top" }}
      triggerClassName="relative z-20 inline-flex items-center"
    >
      {badge}
    </InfoTooltip>
  );
}

function ProductionTooltip({
  isLive,
  isRolledBack,
  wasRolledBack,
  liveSince,
}: {
  isLive: boolean;
  isRolledBack: boolean;
  wasRolledBack: boolean;
  liveSince: number | null;
}) {
  if (wasRolledBack) {
    return (
      <span className="flex flex-col gap-0.5">
        <span className="font-medium">This deployment was rolled back</span>
        <span>Traffic moved back to an earlier deployment.</span>
      </span>
    );
  }

  if (!isLive) {
    return (
      <span className="flex flex-col gap-0.5">
        <span className="font-medium">Production environment</span>
        <span>Not receiving production traffic.</span>
      </span>
    );
  }

  const since =
    liveSince === null ? "Live" : `Live since ${format(liveSince, "d MMM yyyy, HH:mm")}`;

  return (
    <span className="flex flex-col gap-0.5">
      <span className="font-medium">{isRolledBack ? `${since} (rollback)` : since}</span>
      <span>Receiving production traffic.</span>
    </span>
  );
}
