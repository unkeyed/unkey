import type * as React from "react";

import { cn } from "../lib/utils";

export function Meter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex w-full flex-col gap-2", className)} {...props} />;
}

export function MeterHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex items-baseline justify-between gap-4", className)} {...props} />;
}

export function MeterLabel({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) {
  return <span className={cn("text-[13px] text-gray-11", className)} {...props} />;
}

export function MeterValue({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn("font-medium text-[13px] text-gray-12 tabular-nums", className)}
      {...props}
    />
  );
}

export type MeterTrackProps = Omit<React.HTMLAttributes<HTMLDivElement>, "children"> & {
  /** Fill fraction in [0, 1], or null to render the track with no fill. */
  fraction: number | null;
  /** Background class for the fill, e.g. "bg-info-9". The caller owns the tone. */
  fillClassName?: string;
};

export function MeterTrack({ fraction, fillClassName, className, ...props }: MeterTrackProps) {
  const percent = fraction === null ? 0 : Math.min(100, Math.max(0, fraction * 100));

  return (
    <div
      role="meter"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={fraction === null ? undefined : Math.round(percent)}
      className={cn("h-1.5 w-full overflow-hidden rounded-full bg-grayA-3", className)}
      {...props}
    >
      {fraction === null ? null : (
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none",
            fillClassName,
          )}
          style={{ width: `${percent}%` }}
        />
      )}
    </div>
  );
}
