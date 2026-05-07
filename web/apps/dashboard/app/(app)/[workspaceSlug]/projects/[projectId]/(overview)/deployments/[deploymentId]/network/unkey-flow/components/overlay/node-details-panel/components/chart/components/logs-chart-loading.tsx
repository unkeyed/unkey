import { useWaveAnimation } from "@/components/logs/overview-charts/hooks";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

type LogsChartLoadingProps = {
  // Height of the loading skeleton. Must match the sibling chart's height
  // to prevent the enclosing panel from resizing when loading flashes on
  // (e.g. a window-selector change that triggers a refetch without
  // keepPreviousData). Defaults to 50 which matches the bar-chart charts;
  // the network area chart passes its own 160.
  height?: number;
};

export const LogsChartLoading = ({ height = 50 }: LogsChartLoadingProps = {}) => {
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
      <ResponsiveContainer
        height={height}
        className="border-b border-grayA-4"
        width="100%"
        initialDimension={{ width: 1, height: 1 }}
      >
        <BarChart margin={{ top: 0, right: -20, bottom: 0, left: -20 }} barGap={0} data={mockData}>
          <YAxis domain={[0, 1.2]} hide />
          <Bar dataKey="success" fill="hsl(var(--accent-3))" isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
