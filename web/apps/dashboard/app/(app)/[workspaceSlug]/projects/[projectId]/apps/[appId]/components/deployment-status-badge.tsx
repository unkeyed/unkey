"use client";
import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { cn } from "@unkey/ui/src/lib/utils";
import type { IconProps } from "nucleo-ui-outline-18";
import {
  IconBanOutline18,
  IconBoltSlashOutline18,
  IconChartActivityOutline18,
  IconCircleCheckOutline18,
  IconCircleWarningOutline18,
  IconCloudUploadOutline18,
  IconEarthOutline18,
  IconHammer2Outline18,
  IconLayerFrontOutline18,
  IconShieldAlertOutline18,
  IconSparkle3Outline18,
} from "nucleo-ui-outline-18";
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
    icon: IconLayerFrontOutline18,
    label: "Pending",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  starting: {
    icon: IconChartActivityOutline18,
    label: "Starting",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  building: {
    icon: IconHammer2Outline18,
    label: "Building",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  deploying: {
    icon: IconCloudUploadOutline18,
    label: "Deploying",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  network: {
    icon: IconEarthOutline18,
    label: "Assigning Domains",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  finalizing: {
    icon: IconSparkle3Outline18,
    label: "Finalizing",
    bgColor: "bg-linear-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  ready: {
    icon: IconCircleCheckOutline18,
    label: "Ready",
    bgColor: "bg-successA-3",
    textColor: "text-successA-11",
    iconColor: "text-success-11",
  },
  failed: {
    icon: IconCircleWarningOutline18,
    label: "Failed",
    bgColor: "bg-errorA-3",
    textColor: "text-errorA-11",
    iconColor: "text-error-11",
  },
  skipped: {
    icon: IconBanOutline18,
    label: "Skipped",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  awaiting_approval: {
    icon: IconShieldAlertOutline18,
    label: "Awaiting Approval",
    bgColor: "bg-warningA-3",
    textColor: "text-warningA-11",
    iconColor: "text-warning-11",
  },
  stopped: {
    icon: IconBoltSlashOutline18,
    label: "Stopped",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  superseded: {
    icon: IconBanOutline18,
    label: "Superseded",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  cancelled: {
    icon: IconBanOutline18,
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
      <Icon className={cn("size-4", iconColor, animated && "relative z-5")} />
      <span className={cn(textColor, "text-xs", animated && "relative z-5")}>{label}</span>
    </div>
  );
};
