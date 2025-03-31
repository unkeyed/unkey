import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import { useWaveAnimation } from "./hooks";
import type { ChartLabels } from "./types";

type GenericChartLoadingProps = {
  labels: ChartLabels;
  animationSpeed?: number;
  dataPoints?: number;
};

export const OverviewChartLoader = ({
  labels,
  animationSpeed,
  dataPoints,
}: GenericChartLoadingProps) => {
  // Use the wave animation hook with built-in defaults
  const { mockData, currentTime } = useWaveAnimation({
    labels,
    dataPoints,
    animationSpeed,
  });

  return (
    <div className="flex flex-col h-full animate-pulse">
      {/* Header section */}
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">{labels.title}</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
            &nbsp;
          </div>
        </div>
        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.primaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded -w-full">
              &nbsp;
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.secondaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
              &nbsp;
            </div>
          </div>
        </div>
      </div>
      {/* Chart area */}
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <YAxis domain={[0, 1]} hide />
            <Bar
              dataKey={labels.primaryKey}
              fill="hsl(var(--accent-3))"
              stackId="a"
              isAnimationActive={false} // Disable default animation for smoother wave effect
            />
            <Bar
              dataKey={labels.secondaryKey}
              fill="hsl(var(--accent-3))"
              stackId="a"
              isAnimationActive={false} // Disable default animation for smoother wave effect
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
      {/* Time labels footer */}
      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between">
        {calculateTimePoints(currentTime, currentTime).map((time, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <div key={i} className="z-10">
            {formatTimestampLabel(time)}
          </div>
        ))}
      </div>
    </div>
  );
};
