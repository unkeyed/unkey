import { useWaveAnimation } from "@/components/logs/overview-charts/hooks";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

export const LogsChartLoading = () => {
  const { mockData } = useWaveAnimation({
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
      <ResponsiveContainer height={50} className="border-b border-grayA-4" width="100%">
        <BarChart margin={{ top: 0, right: -20, bottom: 0, left: -20 }} barGap={0} data={mockData}>
          <YAxis domain={[0, 1.2]} hide />
          <Bar dataKey="success" fill="hsl(var(--accent-3))" isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
