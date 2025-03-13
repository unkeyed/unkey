import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import { useWaveAnimation } from "../../overview-charts/hooks";
import { calculateTimePoints } from "../utils/calculate-timepoints";
import { formatTimestampLabel } from "../utils/format-timestamp";

export const LogsChartLoading = () => {
  const { mockData, currentTime } = useWaveAnimation({
    dataPoints: 300,
    labels: {
      primaryKey: "success",
      title: "Logs Activity",
      primaryLabel: "Success",
      secondaryLabel: "",
      secondaryKey: "",
    },
  });

  return (
    <div className="w-full relative">
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between">
        {calculateTimePoints(currentTime, currentTime).map((time, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <div key={i} className="z-10">
            {formatTimestampLabel(time)}
          </div>
        ))}
      </div>
      <ResponsiveContainer height={50} className="border-b border-gray-4" width="100%">
        <BarChart margin={{ top: 0, right: -20, bottom: 0, left: -20 }} barGap={0} data={mockData}>
          <YAxis domain={[0, 1.2]} hide />
          <Bar dataKey="success" fill="hsl(var(--accent-3))" isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
