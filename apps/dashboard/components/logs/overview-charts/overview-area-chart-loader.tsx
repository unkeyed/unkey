import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { useEffect, useState } from "react";
import { Area, AreaChart, ResponsiveContainer, YAxis } from "recharts";
import type { TimeseriesChartLabels } from "./overview-area-chart";

type TimeseriesChartLoadingProps = {
  labels: TimeseriesChartLabels;
};

/**
 * Generic loading component for timeseries area chart displays with animated areas
 */
export const OverviewAreaChartLoader = ({ labels }: TimeseriesChartLoadingProps) => {
  const [mockData, setMockData] = useState(() => generateInitialData(labels.metrics));

  function generateInitialData(metrics: TimeseriesChartLabels["metrics"]) {
    return Array.from({ length: 100 }).map((_, index) => {
      const base = Math.sin(index * 0.1) * 0.5 + 1;
      const dataPoint: Record<string, any> = {
        originalTimestamp: Date.now() - (100 - index) * 60000,
      };

      // Add each metric with slightly different values
      metrics.forEach((metric, i) => {
        dataPoint[metric.key] = base * 100 * (1 + i * 0.5);
      });

      return dataPoint;
    });
  }

  useEffect(() => {
    const interval = setInterval(() => {
      setMockData((prevData) =>
        prevData.map((item) => {
          const updatedItem = { ...item };

          // Update each metric value
          labels.metrics.forEach((metric) => {
            updatedItem[metric.key] = item[metric.key] * (0.95 + Math.random() * 0.1);
          });

          return updatedItem;
        }),
      );
    }, 1000);

    return () => clearInterval(interval);
  }, [labels.metrics]);

  const currentTime = Date.now();
  const timePoints = calculateTimePoints(currentTime - 100 * 60000, currentTime);

  return (
    <div className="flex flex-col h-full animate-pulse">
      {/* Header section */}
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">{labels.rangeLabel}</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
            &nbsp;
          </div>
        </div>
        <div className="flex gap-10 items-center">
          {labels.metrics.map((metric) => (
            <div key={metric.key} className="flex flex-col gap-1">
              <div className="flex gap-2 items-center">
                <div className="rounded h-[10px] w-1" style={{ backgroundColor: metric.color }} />
                <div className="text-accent-10 text-[11px] leading-4">{metric.label}</div>
              </div>
              <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
                &nbsp;
              </div>
            </div>
          ))}
        </div>
      </div>
      {/* Chart area */}
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <defs>
              {labels.metrics.map((metric) => (
                <linearGradient
                  key={`${metric.key}Gradient`}
                  id={`${metric.key}Gradient`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="0%" stopColor={metric.color} stopOpacity={0.2} />
                  <stop offset="45%" stopColor={metric.color} stopOpacity={0.1} />
                  <stop offset="100%" stopColor={metric.color} stopOpacity={0.02} />
                </linearGradient>
              ))}
            </defs>
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.1]} hide />

            {labels.metrics.map((metric) => (
              <Area
                key={metric.key}
                type="monotone"
                dataKey={metric.key}
                stroke={metric.color}
                strokeWidth={2}
                fill={`url(#${metric.key}Gradient)`}
                fillOpacity={1}
                isAnimationActive={true}
                animationDuration={800}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>
      {/* Time labels footer */}
      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between border-t-gray-2">
        {timePoints.map((time, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <div key={i} className="z-10">
            {formatTimestampLabel(time)}
          </div>
        ))}
      </div>
    </div>
  );
};
