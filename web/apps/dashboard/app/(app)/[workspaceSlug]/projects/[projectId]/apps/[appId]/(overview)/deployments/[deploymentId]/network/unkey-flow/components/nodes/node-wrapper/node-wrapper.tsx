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
  const isUnhealthy = health === "unhealthy";
  const { ring, glow } = getHealthStyles(health);

  return (
    <div
      className={cn(
        "w-[282px] rounded-[14px]",
        isDisabled
          ? "grayscale opacity-90 cursor-not-allowed"
          : cn(
              "transition-shadow duration-200 ease-out cursor-pointer",
              "hover:ring-2 hover:ring-offset-0",
              ring,
              glow,
              // For unhealthy nodes, also paint a persistent (non-hover)
              // red ring + soft glow so a crashing instance is visible at
              // a glance on the graph rather than only when the cursor
              // happens to be over it.
              isUnhealthy &&
                "ring-2 ring-errorA-5 shadow-[0_0_0_1px_hsl(var(--errorA-5)),0_0_30px_color-mix(in_srgb,hsl(var(--errorA-9))_18%,transparent)]",
            ),
      )}
    >
      <HealthBanner healthStatus={health} />
      <div
        className={cn(
          "w-[282px] h-[100px] rounded-[14px] flex flex-col shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]",
          // Tint the card body itself error-red when unhealthy so it's
          // unmistakable; healthy/syncing/unknown stay on the default
          // white/black background.
          isUnhealthy
            ? "border border-errorA-4 bg-errorA-1 dark:bg-errorA-1"
            : "border border-grayA-4 bg-white dark:bg-black",
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
    unhealthy: {
      ring: "hover:ring-errorA-3",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--errorA-3)),0_0_30px_color-mix(in_srgb,hsl(var(--errorA-9))_20%,transparent)]",
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
