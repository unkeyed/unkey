import type { ChartLabels } from "./types";

type GenericChartErrorProps = {
  labels: ChartLabels;
};

/**
 * Generic error component for chart displays
 */
export const OverviewChartError = ({ labels }: GenericChartErrorProps) => {
  return (
    <div className="flex flex-col h-full">
      {/* Header section matching the main chart */}
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">{labels.title}</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7">--</div>
        </div>
        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.primaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">--</div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.secondaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">--</div>
          </div>
        </div>
      </div>
      {/* Chart area */}
      <div className="flex-1 min-h-0 flex items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <span className="text-sm text-accent-9">Could not retrieve data</span>
        </div>
      </div>
      {/* Time labels footer */}
      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between">
        {Array(5)
          .fill(0)
          .map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            <div key={i} className="z-10">
              --:--
            </div>
          ))}
      </div>
    </div>
  );
};
