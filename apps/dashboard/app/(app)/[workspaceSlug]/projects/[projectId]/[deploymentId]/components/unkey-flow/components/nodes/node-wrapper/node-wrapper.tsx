"use client";

import { cn } from "@/lib/utils";
import type { PropsWithChildren } from "react";
import type { HealthStatus } from "../types";
import { HealthBanner } from "./health-banner";

type NodeWrapperProps = PropsWithChildren<{
  health: HealthStatus;
}>;

export function NodeWrapper({ health, children }: NodeWrapperProps) {
  const isDisabled = health === "disabled";
  const { ring, glow } = getHealthStyles(health);

  return (
    <div
      className={cn(
        "relative w-[282px] rounded-[14px]",
        isDisabled
          ? "grayscale opacity-90 cursor-not-allowed"
          : cn(
              "hover:scale-[1.001] transition-all duration-200 ease-out cursor-pointer",
              "hover:ring-2 hover:ring-offset-0",
              ring,
              glow,
            ),
      )}
    >
      <HealthBanner healthStatus={health} />
      <div
        className={cn(
          "relative z-20 w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]",
        )}
      >
        {children}
      </div>
    </div>
  );
}

function getHealthStyles(health: HealthStatus): { ring: string; glow: string } {
  const styleMap: Record<HealthStatus, { ring: string; glow: string }> = {
    normal: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    unstable: {
      ring: "hover:ring-orangeA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--orangeA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--orangeA-9))_20%,transparent)]",
    },
    degraded: {
      ring: "hover:ring-warningA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--warningA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--warningA-9))_20%,transparent)]",
    },
    unhealthy: {
      ring: "hover:ring-errorA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--errorA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--errorA-9))_20%,transparent)]",
    },
    recovering: {
      ring: "hover:ring-featureA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--featureA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--featureA-9))_20%,transparent)]",
    },
    health_syncing: {
      ring: "hover:ring-infoA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--infoA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--infoA-9))_20%,transparent)]",
    },
    unknown: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
    disabled: {
      ring: "hover:ring-grayA-2",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--grayA-9))_17.5%,transparent)]",
    },
  };
  return styleMap[health];
}
