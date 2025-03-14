import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useEffect, useRef, useState } from "react";
import { Area, AreaChart, ResponsiveContainer, YAxis } from "recharts";
import type { TimeseriesChartLabels } from "./overview-area-chart";

type TimeseriesChartLoadingProps = {
  labels: TimeseriesChartLabels;
};

/**
 * Enhanced loading component for timeseries area chart displays with fluid, animated waves
 */
export const OverviewAreaChartLoader = ({ labels }: TimeseriesChartLoadingProps) => {
  const labelsWithDefaults = {
    ...labels,
    showRightSide: labels.showRightSide !== undefined ? labels.showRightSide : true,
    reverse: labels.reverse !== undefined ? labels.reverse : false,
  };

  const [mockData, setMockData] = useState(() => generateInitialData());
  const [phase, setPhase] = useState(0);
  const animationRef = useRef(0);

  function generateInitialData() {
    return Array.from({ length: 100 }).map((_, index) => {
      const dataPoint: Record<string, any> = {
        index,
        originalTimestamp: Date.now() - (100 - index) * 60000,
      };

      return dataPoint;
    });
  }

  // Animation frame function with smooth, continuous wave patterns
  // biome-ignore lint/correctness/useExhaustiveDependencies: go touch grass biome
  const animate = useCallback(() => {
    setPhase((prev) => prev + 0.01);
    animationRef.current = requestAnimationFrame(animate);
  }, []);

  // Start/stop animation with requestAnimationFrame for smoother performance
  useEffect(() => {
    animationRef.current = requestAnimationFrame(animate);
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [animate]);

  // Update data based on the phase
  useEffect(() => {
    setMockData((prevData) =>
      prevData.map((item, index) => {
        const updatedItem = { ...item };

        // Create flowing wave patterns with varying amplitudes and frequencies per metric
        labelsWithDefaults.metrics.forEach((metric, metricIndex) => {
          // Primary wave with slowly moving amplitude
          const primaryAmplitude = 0.6 + Math.sin(phase * 0.2) * 0.3;

          // Create unique wave pattern for each metric
          const baseWave = Math.sin(-phase + index * 0.1) * primaryAmplitude;
          const secondWave = Math.cos(-phase * 0.7 + index * 0.08) * 0.3;
          const thirdWave = Math.sin(-phase * 0.5 + index * 0.15) * 0.2;

          // Combine waves with different weights per metric
          const combinedWave =
            baseWave * (1 - metricIndex * 0.2) +
            secondWave * (0.5 + metricIndex * 0.2) +
            thirdWave * (0.3 + metricIndex * 0.1);

          // Scale the wave to appropriate values with increasing baseline per metric
          updatedItem[metric.key] = (combinedWave + 2) * 100 * (1 + metricIndex * 0.5);
        });

        return updatedItem;
      }),
    );
  }, [phase, labelsWithDefaults.metrics]);

  const currentTime = Date.now();
  const timePoints = calculateTimePoints(currentTime - 100 * 60000, currentTime);

  return (
    <div className="flex flex-col h-full animate-pulse">
      {/* Header section with support for reverse layout */}
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
          <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
            &nbsp;
          </div>
        </div>

        {/* Right side section shown conditionally */}
        {labelsWithDefaults.showRightSide && (
          <div className="flex gap-10 items-center">
            {labelsWithDefaults.metrics.map((metric) => (
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
        )}
      </div>

      {/* Chart area */}
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <defs>
              {labelsWithDefaults.metrics.map((metric) => (
                <linearGradient
                  key={`${metric.key}Gradient`}
                  id={`${metric.key}Gradient`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="0%" stopColor={metric.color} stopOpacity={0.3} />
                  <stop offset="60%" stopColor={metric.color} stopOpacity={0.1} />
                  <stop offset="100%" stopColor={metric.color} stopOpacity={0.02} />
                </linearGradient>
              ))}
            </defs>
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.1]} hide />

            {labelsWithDefaults.metrics.map((metric) => (
              <Area
                key={metric.key}
                type="monotone"
                dataKey={metric.key}
                stroke={metric.color}
                strokeWidth={1.5}
                fill={`url(#${metric.key}Gradient)`}
                fillOpacity={1}
                isAnimationActive={false} // Disable recharts animation to use our custom animation
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>

      {/* Time labels footer */}
      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between">
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
