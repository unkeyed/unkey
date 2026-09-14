"use client";

import { ENVIRONMENT_KIND, type Environment } from "@/lib/collections/deploy/environments";
import { cn } from "@/lib/utils";
import { ArrowDotAntiClockwise, CircleXMark, Cloud, Eye } from "@unkey/icons";
import { match } from "@unkey/match";
import { InfoTooltip } from "@unkey/ui";
import { format } from "date-fns";

export type EnvironmentBadgeRollout = "none" | "live" | "liveAfterRollback" | "rolledBackFrom";

type EnvironmentBadgeProps = {
  environment: Environment;
  rollout: EnvironmentBadgeRollout;
  liveSince?: number | null;
};

const BASE_CLASS =
  "inline-flex h-5.5 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs leading-none";
const OUTLINED_CLASS = "border-grayA-5 text-accent-12";
const LIVE_CLASS = "border-transparent bg-info-11 text-white dark:bg-info-9 dark:text-gray-1";
const ROLLED_BACK_FROM_CLASS = "border-transparent bg-errorA-3 text-error-11";

export function EnvironmentBadge({
  environment,
  rollout,
  liveSince = null,
}: EnvironmentBadgeProps) {
  if (environment.kind !== ENVIRONMENT_KIND.production) {
    return (
      <span className={cn(BASE_CLASS, OUTLINED_CLASS)}>
        <Eye iconSize="sm-regular" className="shrink-0" />
        <span className="capitalize">{environment.slug}</span>
      </span>
    );
  }

  const liveLabel =
    liveSince === null ? "Live" : `Live since ${format(liveSince, "d MMM yyyy, HH:mm")}`;

  const { Icon, className, title, detail } = match(rollout)
    .with("none", () => ({
      Icon: Cloud,
      className: OUTLINED_CLASS,
      title: "Production environment",
      detail: "Not receiving production traffic.",
    }))
    .with("live", () => ({
      Icon: Cloud,
      className: LIVE_CLASS,
      title: liveLabel,
      detail: "Receiving production traffic.",
    }))
    .with("liveAfterRollback", () => ({
      Icon: ArrowDotAntiClockwise,
      className: LIVE_CLASS,
      title: `${liveLabel} (rollback)`,
      detail: "Receiving production traffic.",
    }))
    .with("rolledBackFrom", () => ({
      Icon: CircleXMark,
      className: ROLLED_BACK_FROM_CLASS,
      title: "This deployment was rolled back",
      detail: "Traffic moved back to an earlier deployment.",
    }))
    .exhaustive();

  return (
    <InfoTooltip
      content={
        <span className="flex flex-col gap-0.5">
          <span className="font-medium">{title}</span>
          <span>{detail}</span>
        </span>
      }
      variant="inverted"
      position={{ side: "top" }}
      triggerClassName="relative z-20 inline-flex items-center"
    >
      <span className={cn(BASE_CLASS, className)}>
        <Icon iconSize="sm-regular" className="shrink-0" />
        <span className="capitalize">{environment.slug}</span>
      </span>
    </InfoTooltip>
  );
}
