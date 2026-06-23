"use client";

import { cn } from "@/lib/utils";

type UsageMeterProps = {
  label: string;
  /** Right-aligned value text, e.g. "$46.20 of $100.00" or "32,400 / 150,000". */
  value: string;
  /** Fill fraction in [0, 1], or null to render the track without a fill. */
  fraction: number | null;
  /** Tailwind background class for the fill, e.g. "bg-orange-9". */
  fillClassName: string;
};

/**
 * Flat linear usage bar: label and value on one line, a thin track below.
 * The fill is clamped so over-quota usage pins at 100% instead of overflowing.
 */
export const UsageMeter: React.FC<UsageMeterProps> = ({
  label,
  value,
  fraction,
  fillClassName,
}) => {
  const percent = fraction === null ? 0 : Math.min(100, Math.max(0, fraction * 100));

  return (
    <div className="flex w-full flex-col gap-2">
      <div className="flex items-baseline justify-between gap-4">
        <span className="text-[13px] text-gray-11">{label}</span>
        <span className="font-medium text-[13px] text-gray-12 tabular-nums">{value}</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-grayA-3">
        {fraction !== null ? (
          <div
            className={cn("h-full rounded-full transition-[width] duration-300", fillClassName)}
            style={{ width: `${percent}%` }}
          />
        ) : null}
      </div>
    </div>
  );
};
