import { cn } from "@/lib/utils";
import { ResponsiveContainer } from "recharts";
import type { ChartErrorProps } from "./types";

/**
 * Unified chart error component that handles three display variants:
 *
 * - "simple": Minimal centered error message (for stats cards)
 * - "compact": Error with time label placeholders and fixed height (for logs charts)
 * - "full": Complete layout with header, metrics, and footer (for overview area charts)
 *
 * @example
 * // Simple variant (stats card)
 * <ChartError variant="simple" message="Could not retrieve logs" />
 *
 * @example
 * // Compact variant (logs chart)
 * <ChartError variant="compact" height={50} />
 *
 * @example
 * // Full variant (overview charts)
 * <ChartError
 *   variant="full"
 *   labels={{
 *     rangeLabel: "Last 24h",
 *     metrics: [{ key: "success", label: "Success", color: "#10b981" }]
 *   }}
 * />
 */
export const ChartError = ({
  variant = "simple",
  message = "Could not retrieve data",
  labels,
  height = 50,
  className,
}: ChartErrorProps) => {
  // Simple variant: just centered error message
  if (variant === "simple") {
    return (
      <div className={cn("flex flex-col h-full", className)}>
        <div className="flex-1 min-h-0 flex items-center justify-center">
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm text-accent-9">{message}</span>
          </div>
        </div>
      </div>
    );
  }

  // Compact variant: with time placeholders and fixed height
  if (variant === "compact") {
    return (
      <div className={cn("w-full relative", className)}>
        <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between opacity-50">
          {Array(5)
            .fill(0)
            .map((_, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: static placeholder array
              <div key={i} className="z-10">
                --:--
              </div>
            ))}
        </div>
        <ResponsiveContainer height={height} className="border-b border-gray-4" width="100%">
          <div className="flex-1 flex items-center justify-center h-full">
            <div className="flex flex-col items-center gap-2">
              <span className="text-sm text-accent-9">{message}</span>
            </div>
          </div>
        </ResponsiveContainer>
      </div>
    );
  }

  // Full variant: complete layout with header, metrics, and footer
  if (variant === "full" && labels) {
    const labelsWithDefaults = {
      ...labels,
      showRightSide: labels.showRightSide !== undefined ? labels.showRightSide : true,
      reverse: labels.reverse !== undefined ? labels.reverse : false,
    };

    return (
      <div className={cn("flex flex-col h-full", className)}>
        {/* Header section with metrics */}
        <div
          className={cn(
            "pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10",
            labelsWithDefaults.reverse && "flex-row-reverse",
          )}
        >
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              {labelsWithDefaults.reverse &&
                labelsWithDefaults.metrics.map((metric) => (
                  <div
                    key={metric.key}
                    className="rounded h-[10px] w-1"
                    style={{ backgroundColor: metric.color }}
                  />
                ))}
              <div className="text-accent-10 text-[11px] leading-4">
                {labelsWithDefaults.rangeLabel}
              </div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">--</div>
          </div>

          {/* Right side section shown conditionally */}
          {labelsWithDefaults.showRightSide && (
            <div className="flex gap-10 items-center">
              {labelsWithDefaults.metrics.map((metric) => (
                <div key={metric.key} className="flex flex-col gap-1">
                  <div className="flex gap-2 items-center">
                    <div
                      className="rounded h-[10px] w-1"
                      style={{ backgroundColor: metric.color }}
                    />
                    <div className="text-accent-10 text-[11px] leading-4">{metric.label}</div>
                  </div>
                  <div className="text-accent-12 text-[18px] font-semibold leading-7">--</div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Chart area with error message */}
        <div className="flex-1 min-h-0 flex items-center justify-center">
          <div className="flex flex-col items-center gap-2">
            <span className="text-sm text-accent-9">{message}</span>
          </div>
        </div>

        {/* Time labels footer */}
        <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between">
          {Array(5)
            .fill(0)
            .map((_, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: static placeholder array
              <div key={i} className="z-10">
                --:--
              </div>
            ))}
        </div>
      </div>
    );
  }

  // Fallback to simple if variant is "full" but no labels provided
  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="flex-1 min-h-0 flex items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <span className="text-sm text-accent-9">{message}</span>
        </div>
      </div>
    </div>
  );
};
