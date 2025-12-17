import { useWaveAnimation } from "@/components/logs/overview-charts/hooks";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

export const LogsChartLoading = () => {
  // Use our custom hook with both primary and secondary keys
  const { mockData } = useWaveAnimation({
    animate: false,
    dataPoints: 100,
    labels: {
      primaryKey: "success",
      title: "Logs Activity",
      primaryLabel: "Success",
      secondaryLabel: "",
      secondaryKey: "",
    },
  });

  return (
    <div className="flex flex-col h-full animate-pulse">
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <YAxis domain={[0, 1]} hide />
            <Bar
              dataKey="success"
              fill="hsl(var(--accent-3))"
              stackId="a"
              isAnimationActive={false}
            />
            <Bar
              dataKey="error"
              fill="hsl(var(--accent-3))"
              stackId="a"
              isAnimationActive={false}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
