import type { ChartConfig } from "@/components/ui/chart";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

type LogsChartEmptyProps = {
  config: ChartConfig;
  height: number;
};

export const LogsChartEmpty = ({ config, height }: LogsChartEmptyProps) => {
  const placeholderData = Array.from({ length: 30 }, (_, i) => ({
    index: i,
    value: 0.1,
  }));

  const dataKey = Object.keys(config)[0];
  const color = config[dataKey]?.color || "hsl(var(--gray-6))";

  return (
    <div className="w-full relative">
      <ResponsiveContainer height={height} className="border-b border-grayA-4" width="100%">
        <BarChart
          data={placeholderData}
          margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
          barCategoryGap={0.5}
        >
          <YAxis domain={[0, 1]} hide />
          <Bar dataKey="value" fill={color} opacity={0.15} isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};
