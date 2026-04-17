"use client";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import {
  Ban,
  CircleCheck,
  CircleHalfDottedClock,
  CircleWarning,
  CloudUp,
  Earth,
  Hammer2,
  LayerFront,
  Pulse,
  ShieldAlert,
  Sparkle3,
} from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { cn } from "@unkey/ui/src/lib/utils";
import type { FC } from "react";

type StatusConfig = {
  icon: FC<IconProps>;
  label: string;
  bgColor: string;
  textColor: string;
  iconColor: string;
  animated?: boolean;
};

const statusConfigs: Record<DeploymentStatus, StatusConfig> = {
  pending: {
    icon: LayerFront,
    label: "Pending",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  starting: {
    icon: Pulse,
    label: "Starting",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  building: {
    icon: Hammer2,
    label: "Building",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  deploying: {
    icon: CloudUp,
    label: "Deploying",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  network: {
    icon: Earth,
    label: "Assigning Domains",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  finalizing: {
    icon: Sparkle3,
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
    label: "Failed",
    bgColor: "bg-errorA-3",
    textColor: "text-errorA-11",
    iconColor: "text-error-11",
  },
  skipped: {
    icon: Ban,
    label: "Skipped",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  awaiting_approval: {
    icon: ShieldAlert,
    label: "Awaiting Approval",
    bgColor: "bg-warningA-3",
    textColor: "text-warningA-11",
    iconColor: "text-warning-11",
  },
  stopped: {
    icon: CircleHalfDottedClock,
    label: "Stopped",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  superseded: {
    icon: Ban,
    label: "Superseded",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  cancelled: {
    icon: Ban,
    label: "Cancelled",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
};

type DeploymentStatusBadgeProps = {
  status: DeploymentStatus;
  className?: string;
};

export const DeploymentStatusBadge = ({ status, className }: DeploymentStatusBadgeProps) => {
  const config = statusConfigs[status];

  if (!config) {
    throw new Error(`Invalid deployment status: ${status}`);
  }

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
      <Icon iconSize="md-regular" className={cn(iconColor, animated && "relative z-5")} />
      <span className={cn(textColor, "text-xs", animated && "relative z-5")}>{label}</span>
    </div>
  );
};
