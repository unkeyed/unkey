import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import { calculateTimePoints } from "../utils/calculate-timepoints";
import { formatTimestampLabel } from "../utils/format-timestamp";

export const LogsChartLoading = () => {
  const mockData = Array.from({ length: 100 }).map(() => ({
    success: Math.random() * 0.5 + 0.5, // Random values between 0.5 and 1
  }));

  return (
    <div className="w-full relative animate-pulse">
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between">
        {calculateTimePoints(Date.now(), Date.now()).map((time, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: it's safe to use index here
          <div key={i} className="z-10">
            {formatTimestampLabel(time)}
          </div>
        ))}
      </div>
      <ResponsiveContainer height={50} className="border-b border-gray-4" width="100%">
        <BarChart margin={{ top: 0, right: -20, bottom: 0, left: -20 }} barGap={0} data={mockData}>
          <YAxis domain={[0, (dataMax: number) => dataMax * 2]} hide />
          <Bar dataKey="success" fill="hsl(var(--accent-3))" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
