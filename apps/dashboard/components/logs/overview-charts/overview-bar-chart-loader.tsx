import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { useEffect, useState } from "react";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import type { ChartLabels } from "./types";

type GenericChartLoadingProps = {
  labels: ChartLabels;
};

const barCount = 100;
/**
 * Generic loading component for chart displays with wave-like animated bars
 */
export const OverviewChartLoader = ({ labels }: GenericChartLoadingProps) => {
  const [mockData, setMockData] = useState(generateInitialData());
  const [phase, setPhase] = useState(0);

  function generateInitialData() {
    return Array.from({ length: barCount }).map((_, index) => ({
      [labels.primaryKey]: 0.5,
      [labels.secondaryKey]: 0.2,
      index,
      originalTimestamp: Date.now(),
    }));
  }

  useEffect(() => {
    const interval = setInterval(() => {
      setPhase((prevPhase) => prevPhase + 0.05);
    }, 40); // Update very frequently for extra smoothness, but with smaller increments

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    setMockData((prevData) =>
      prevData.map((item, index) => {
        const wavePosition = -phase + index * (Math.PI / 20);

        const primaryWave = Math.sin(wavePosition) * 0.2 + 0.5;
        const secondaryWave = Math.sin(wavePosition * 1.2) * 0.12 + 0.25;

        return {
          ...item,
          [labels.primaryKey]: primaryWave,
          [labels.secondaryKey]: secondaryWave,
        };
      }),
    );
  }, [phase, labels.primaryKey, labels.secondaryKey]);

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
              animationDuration={100}
              isAnimationActive={false} // Disable default animation for smoother wave effect
            />
            <Bar
              dataKey={labels.secondaryKey}
              fill="hsl(var(--accent-3))"
              stackId="a"
              animationDuration={100}
              isAnimationActive={false} // Disable default animation for smoother wave effect
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
      {/* Time labels footer */}
      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between border-t-gray-2">
        {calculateTimePoints(Date.now(), Date.now()).map((time, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <div key={i} className="z-10">
            {formatTimestampLabel(time)}
          </div>
        ))}
      </div>
    </div>
  );
};
