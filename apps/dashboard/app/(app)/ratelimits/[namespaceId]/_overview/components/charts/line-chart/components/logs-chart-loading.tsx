import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { useEffect, useState } from "react";
import { Area, AreaChart, ResponsiveContainer, YAxis } from "recharts";

type MockDataPoint = {
  avgLatency: number;
  p99Latency: number;
  originalTimestamp: number;
};

function generateInitialData(): MockDataPoint[] {
  return Array.from({ length: 100 }).map((_, index) => {
    const base = Math.sin(index * 0.1) * 0.5 + 1;
    return {
      avgLatency: base * 100,
      p99Latency: base * 150,
      originalTimestamp: Date.now() - (100 - index) * 60000,
    };
  });
}

export const LogsTimeseriesAreaChartLoading = () => {
  const [mockData, setMockData] = useState<MockDataPoint[]>(generateInitialData());

  useEffect(() => {
    const interval = setInterval(() => {
      setMockData((prevData) =>
        prevData.map((item) => ({
          ...item,
          avgLatency: item.avgLatency * (0.95 + Math.random() * 0.1),
          p99Latency: item.p99Latency * (0.95 + Math.random() * 0.1),
        })),
      );
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const currentTime = Date.now();
  const timePoints = calculateTimePoints(currentTime - 100 * 60000, currentTime);

  return (
    <div className="flex flex-col h-full animate-pulse">
      {/* Header section */}
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">DURATION</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
            &nbsp;
          </div>
        </div>

        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">AVG</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
              &nbsp;
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">P99</div>
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
          <AreaChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="avgGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="hsl(var(--accent-11))" stopOpacity={0.2} />
                <stop offset="45%" stopColor="hsl(var(--accent-11))" stopOpacity={0.1} />
                <stop offset="100%" stopColor="hsl(var(--accent-11))" stopOpacity={0.02} />
              </linearGradient>
              <linearGradient id="p99Gradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="hsl(var(--warning-10))" stopOpacity={0.2} />
                <stop offset="45%" stopColor="hsl(var(--warning-10))" stopOpacity={0.1} />
                <stop offset="100%" stopColor="hsl(var(--warning-10))" stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.1]} hide />
            <Area
              type="monotone"
              dataKey="avgLatency"
              stroke="hsl(var(--accent-11))"
              strokeWidth={2}
              fillOpacity={0}
              isAnimationActive={true}
              animationDuration={800}
            />
            <Area
              type="monotone"
              dataKey="p99Latency"
              stroke="hsl(var(--warning-10))"
              strokeWidth={2}
              fill="url(#p99Gradient)"
              fillOpacity={1}
              isAnimationActive={true}
              animationDuration={800}
            />
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
