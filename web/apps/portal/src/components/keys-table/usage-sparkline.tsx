import { useMemo } from "react";
import { cn } from "~/lib/utils";

type Props = {
  buckets: number[];
  errors?: number[];
  maxBars?: number;
  ariaLabel?: string;
};

const MAX_HEIGHT_BUFFER_FACTOR = 1.3;
const MAX_BAR_HEIGHT = 28;

type Bar = {
  key: string;
  top: number;
  bottom: number;
};

/**
 * Inline bar chart matching the dashboard's VerificationBarChart visual spec.
 * 30 fixed-width bars, each stacked as error segment (top, red) + valid segment
 * (bottom, gray). Heights in pixels, scaled to the tallest bucket with a 1.3×
 * buffer so the peak sits at ~77% of the container.
 *
 * Pure presentational — takes already-shaped arrays, no data fetching or
 * caching. Dashboard's production version (`web/apps/dashboard/components/
 * api-keys-table/components/bar-chart/`) does fetch + cache + tooltip; we skip
 * all of that for the portal prototype.
 */
export function UsageSparkline({ buckets, errors, maxBars = 30, ariaLabel }: Props) {
  const bars = useMemo((): Bar[] => {
    const recent = buckets.slice(-maxBars);
    const recentErrors = errors?.slice(-maxBars) ?? [];
    const maxTotal = Math.max(...recent, 1) * MAX_HEIGHT_BUFFER_FACTOR;

    const filled: Bar[] = recent.map((total, i) => {
      const err = recentErrors[i] ?? 0;
      const totalHeight = Math.min(Math.round((total / maxTotal) * MAX_BAR_HEIGHT), MAX_BAR_HEIGHT);
      const top = err > 0 && total > 0 ? Math.max(Math.round((err / total) * totalHeight), 1) : 0;
      const bottom = Math.max(totalHeight - top, 0);
      return { key: `b${i}`, top, bottom };
    });

    // Pad with empty bars at the start if fewer than maxBars
    while (filled.length < maxBars) {
      filled.unshift({ key: `pad${filled.length}`, top: 0, bottom: 0 });
    }
    return filled;
  }, [buckets, errors, maxBars]);

  return (
    <div
      role="img"
      aria-label={ariaLabel ?? `Usage for the last ${maxBars} hours`}
      className={cn(
        "grid h-[28px] w-[158px] items-end overflow-hidden rounded-t border border-transparent bg-grayA-2 px-1 py-0",
        "transition-colors hover:rounded-md hover:border-grayA-2",
      )}
      style={{
        gridTemplateColumns: `repeat(${maxBars}, 3px)`,
        gap: "2px",
      }}
    >
      {bars.map((bar) => (
        <div key={bar.key} className="flex flex-col">
          <div className="w-[3px] bg-error-9" style={{ height: `${bar.top}px` }} />
          <div className="w-[3px] bg-grayA-5" style={{ height: `${bar.bottom}px` }} />
        </div>
      ))}
    </div>
  );
}
