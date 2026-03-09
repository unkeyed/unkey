import {
  ArrowDotAntiClockwise,
  CircleCheck,
  CircleHalfDottedClock,
  CircleWarning,
  HalfDottedCirclePlay,
  Nut,
} from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { cn } from "@unkey/ui/src/lib/utils";
import type { FC } from "react";

export type DeploymentStatus =
  | "pending"
  | "starting"
  | "building"
  | "deploying"
  | "network"
  | "ready"
  | "finalizing"
  | "failed";

type StatusConfig = {
  icon: FC<IconProps>;
  label: string;
  bgColor: string;
  textColor: string;
  iconColor: string;
  animated?: boolean;
};

const STATUS_CONFIG: Record<DeploymentStatus, StatusConfig> = {
  pending: {
    icon: CircleHalfDottedClock,
    label: "Queued",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  starting: {
    icon: HalfDottedCirclePlay,
    label: "Starting",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  building: {
    icon: Nut,
    label: "Building",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  deploying: {
    icon: HalfDottedCirclePlay,
    label: "Deploying",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  network: {
    icon: ArrowDotAntiClockwise,
    label: "Assigning Domains",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  finalizing: {
    icon: Nut,
    label: "Finalizing",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  ready: {
    icon: CircleCheck,
    label: "Ready",
    bgColor: "bg-successA-3",
    textColor: "text-successA-11",
    iconColor: "text-success-11",
  },
  failed: {
    icon: CircleWarning,
    label: "Error",
    bgColor: "bg-errorA-3",
    textColor: "text-errorA-11",
    iconColor: "text-error-11",
  },
};

type Props = {
  status?: DeploymentStatus;
  className?: string;
};

export const DeploymentStatusBadge = ({ status, className }: Props) => {
  if (!status) {
    throw new Error(`Invalid deployment status: ${status}`);
  }

  const config = STATUS_CONFIG[status];
  const { icon: Icon, label, bgColor, textColor, iconColor, animated } = config;

  return (
    <div
      className={cn(
        "items-center flex gap-2 p-1.5 rounded-md w-fit relative h-5.5",
        animated && "overflow-hidden",
        bgColor,
        className,
      )}
    >
      {animated && (
        <div className="absolute inset-0 bg-linear-to-r from-transparent via-white/40 to-transparent w-[150%] animate-shimmer" />
      )}
      <Icon
        iconSize={config.icon === Nut ? "md-bold" : "md-regular"}
        className={cn(iconColor, animated && "relative z-5")}
      />
      <span className={cn(textColor, "text-xs", animated && "relative z-5")}>{label}</span>
    </div>
  );
};
