import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { useEffect, useState } from "react";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

export const LogsChartLoading = () => {
  const [mockData, setMockData] = useState(generateInitialData());

  function generateInitialData() {
    return Array.from({ length: 100 }).map(() => ({
      success: Math.random() * 0.5 + 0.5,
      error: Math.random() * 0.3,
      originalTimestamp: Date.now(),
    }));
  }

  useEffect(() => {
    const interval = setInterval(() => {
      setMockData((prevData) =>
        prevData.map((item) => ({
          ...item,
          success: Math.random() * 0.5 + 0.5,
          error: Math.random() * 0.3,
        })),
      );
    }, 600); // Update every 200ms for smooth animation

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex flex-col h-full animate-pulse">
      {/* Header section */}
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">REQUESTS</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded w-full">
            &nbsp;
          </div>
        </div>

        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">PASSED</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7 bg-accent-4 rounded -w-full">
              &nbsp;
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">BLOCKED</div>
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
            <YAxis domain={[0, (dataMax: number) => dataMax * 2]} hide />
            <Bar
              dataKey="success"
              fill="hsl(var(--accent-3))"
              stackId="a"
              animationDuration={600}
              isAnimationActive
            />
            <Bar
              dataKey="error"
              fill="hsl(var(--accent-3))"
              stackId="a"
              animationDuration={600}
              isAnimationActive
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
