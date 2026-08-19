"use client";

import { Meter as MeterPrimitive } from "@base-ui/react/meter";
import type React from "react";

import { cn } from "../lib/utils";

const layoutClassName = {
  stacked: "flex-col gap-2",
  inline:
    "flex-row items-center gap-3 [&_[data-slot=meter-track]]:min-w-0 [&_[data-slot=meter-track]]:flex-1",
} as const;

export type MeterProps = MeterPrimitive.Root.Props & {
  layout?: keyof typeof layoutClassName;
};

export function Meter({ className, children, layout = "stacked", value, ...props }: MeterProps) {
  return (
    <MeterPrimitive.Root
      value={value}
      data-layout={layout}
      data-empty={value <= (props.min ?? 0) ? "" : undefined}
      className={cn("group/meter flex w-full", layoutClassName[layout], className)}
      {...props}
    >
      {children ? (
        children
      ) : (
        <MeterTrack>
          <MeterIndicator />
        </MeterTrack>
      )}
    </MeterPrimitive.Root>
  );
}

export function MeterHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex items-baseline justify-between gap-4", className)}
      data-slot="meter-header"
      {...props}
    />
  );
}

export function MeterLabel({ className, ...props }: MeterPrimitive.Label.Props) {
  return (
    <MeterPrimitive.Label
      className={cn("text-[13px] text-gray-11", className)}
      data-slot="meter-label"
      {...props}
    />
  );
}

export function MeterTrack({ className, ...props }: MeterPrimitive.Track.Props) {
  return (
    <MeterPrimitive.Track
      className={cn("block h-1.5 w-full overflow-hidden rounded-full bg-grayA-3", className)}
      data-slot="meter-track"
      {...props}
    />
  );
}

export function MeterIndicator({ className, ...props }: MeterPrimitive.Indicator.Props) {
  return (
    <MeterPrimitive.Indicator
      className={cn(
        "min-w-1 bg-gray-12 transition-[width] duration-300 group-data-[empty]/meter:min-w-0 motion-reduce:transition-none",
        className,
      )}
      data-slot="meter-indicator"
      {...props}
    />
  );
}

export function MeterValue({ className, ...props }: MeterPrimitive.Value.Props) {
  return (
    <MeterPrimitive.Value
      className={cn("font-medium text-[13px] text-gray-12 tabular-nums", className)}
      data-slot="meter-value"
      {...props}
    />
  );
}

export { MeterPrimitive };
