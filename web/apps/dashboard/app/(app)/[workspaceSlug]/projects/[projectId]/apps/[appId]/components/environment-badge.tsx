"use client";

import { ENVIRONMENT_KIND, type Environment, useLiveTarget } from "@/lib/deploy-data";
import {
  IconArrowDotRotateAnticlockwiseOutline12,
  IconCircleXmarkOutline12,
  IconCloudOutline12,
  IconEyeOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { InfoTooltip, Loading, formatTimestamp } from "@unkey/ui";
import { cn } from "cn";

export type EnvironmentBadgeRollout =
  | "none"
  | "live"
  | "liveAfterRollback"
  | "rolledBackFrom"
  | "goingLive"
  | "switchingAway";

// Which side of an unfinished promote or rollback a deployment is on
export type LiveSwitch = "incoming" | "outgoing" | null;

// Marks the rows on each side of a promote or rollback until the switch lands
export function useLiveSwitch(
  currentDeploymentId: string | undefined,
  isRolledBack: boolean,
): (deploymentId: string) => LiveSwitch {
  const liveTarget = useLiveTarget();
  const landed =
    liveTarget === null ||
    (currentDeploymentId === liveTarget.deploymentId && isRolledBack === liveTarget.rolledBack);
  return (deploymentId) => {
    if (landed) {
      return null;
    }
    if (deploymentId === currentDeploymentId) {
      return "outgoing";
    }
    return deploymentId === liveTarget.deploymentId ? "incoming" : null;
  };
}

export function badgeRollout({
  isCurrent,
  isRolledBack,
  isRolledBackFrom,
  liveSwitch = null,
}: {
  isCurrent: boolean;
  isRolledBack: boolean;
  isRolledBackFrom: boolean;
  liveSwitch?: LiveSwitch;
}): EnvironmentBadgeRollout {
  if (liveSwitch === "incoming") {
    return "goingLive";
  }
  if (liveSwitch === "outgoing") {
    return "switchingAway";
  }
  if (isRolledBackFrom) {
    return "rolledBackFrom";
  }
  if (!isCurrent) {
    return "none";
  }
  return isRolledBack ? "liveAfterRollback" : "live";
}

type EnvironmentBadgeProps = {
  environment: Environment;
  rollout: EnvironmentBadgeRollout;
  liveSince?: number | null;
};

const BASE_CLASS =
  "inline-flex h-5.5 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 text-xs leading-none";
const OUTLINED_CLASS = "text-gray-12";
const LIVE_CLASS = "border-info-7 bg-info-3 font-medium text-info-12 dark:bg-info-4";
const ROLLED_BACK_FROM_CLASS = "border-transparent bg-errorA-3 text-error-11";
const GOING_LIVE_CLASS = "border-info-7 text-info-11";

export const LIVE_BADGE_CLASS = cn(BASE_CLASS, LIVE_CLASS);

export function EnvironmentBadge({
  environment,
  rollout,
  liveSince = null,
}: EnvironmentBadgeProps) {
  if (environment.kind !== ENVIRONMENT_KIND.production) {
    return (
      <span className={cn(BASE_CLASS, OUTLINED_CLASS)}>
        <IconEyeOutline12 className="shrink-0" />
        <span className="capitalize">{environment.slug}</span>
      </span>
    );
  }

  const liveLabel =
    liveSince === null ? "Live" : `Live since ${formatTimestamp(liveSince, "MMM d, yyyy, h:mm a")}`;

  const { Icon, className, title, detail } = match(rollout)
    .with("none", () => ({
      Icon: IconCloudOutline12,
      className: OUTLINED_CLASS,
      title: "Production environment",
      detail: "Not receiving production traffic.",
    }))
    .with("live", () => ({
      Icon: IconCloudOutline12,
      className: LIVE_CLASS,
      title: liveLabel,
      detail: "Receiving production traffic.",
    }))
    .with("liveAfterRollback", () => ({
      Icon: IconArrowDotRotateAnticlockwiseOutline12,
      className: LIVE_CLASS,
      title: `${liveLabel} (rollback)`,
      detail: "Receiving production traffic.",
    }))
    .with("rolledBackFrom", () => ({
      Icon: IconCircleXmarkOutline12,
      className: ROLLED_BACK_FROM_CLASS,
      title: "This deployment was rolled back",
      detail: "Traffic moved back to an earlier deployment.",
    }))
    .with("goingLive", () => ({
      Icon: SwitchingIcon,
      className: GOING_LIVE_CLASS,
      title: "Going live",
      detail: "Production traffic is moving to this deployment.",
    }))
    .with("switchingAway", () => ({
      Icon: SwitchingIcon,
      className: LIVE_CLASS,
      title: "Switching production traffic",
      detail: "Serves production traffic until the switch finishes.",
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
      position={{ side: "top" }}
      triggerClassName="relative z-20 inline-flex items-center"
    >
      <span className={cn(BASE_CLASS, className)}>
        <Icon className="shrink-0" />
        <span className="capitalize">{environment.slug}</span>
      </span>
    </InfoTooltip>
  );
}

function SwitchingIcon({ className }: { className?: string }) {
  return <Loading size={12} className={className} />;
}
