"use client";
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
import type { DeploymentStatus } from "../../../filters.schema";

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
    icon: CircleHalfDottedClock,
    label: "Pending",
    bgColor: "bg-grayA-3",
    textColor: "text-grayA-11",
    iconColor: "text-gray-11",
  },
  building: {
    icon: Nut,
    label: "Building",
    bgColor: "bg-gradient-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  deploying: {
    icon: HalfDottedCirclePlay,
    label: "Deploying",
    bgColor: "bg-gradient-to-r from-infoA-5 to-transparent",
    textColor: "text-infoA-11",
    iconColor: "text-info-11",
    animated: true,
  },
  network: {
    icon: ArrowDotAntiClockwise,
    label: "Assigning Domains",
    bgColor: "bg-gradient-to-r from-infoA-5 to-transparent",
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
        "items-center flex gap-2 p-1.5 rounded-md w-fit relative",
        animated && "overflow-hidden",
        bgColor,
        className,
      )}
    >
      {animated && (
        <div
          className="absolute inset-0 bg-gradient-to-r from-transparent via-white/40 to-transparent w-[150%]"
          style={{
            animation: "shimmer 1.2s ease-in-out infinite",
          }}
        />
      )}
      <Icon
        size={config.icon === Nut ? "md-bold" : "md-regular"}
        className={cn(iconColor, animated && "relative z-5")}
      />
      <span className={cn(textColor, "text-xs", animated && "relative z-5")}>{label}</span>

      {animated && (
        <style jsx>{`
          @keyframes shimmer {
            0% {
              transform: translateX(-100%);
            }
            100% {
              transform: translateX(100%);
            }
          }
        `}</style>
      )}
    </div>
  );
};
