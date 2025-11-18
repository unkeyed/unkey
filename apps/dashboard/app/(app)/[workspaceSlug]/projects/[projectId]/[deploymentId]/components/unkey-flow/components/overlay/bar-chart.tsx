import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Bar, BarChart } from "recharts";

type ChartMetricHeaderProps = {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
};

type ChartBarInteractiveProps = {
  header: ChartMetricHeaderProps;
  data: { x: number; y: number }[];
  color: string;
  tooltipFormatter?: (value: number) => string;
  xAxisFormatter?: (timestamp: number) => string;
};

export function ChartBarInteractive({
  header,
  data,
  color,
  tooltipFormatter,
  xAxisFormatter,
}: ChartBarInteractiveProps) {
  const chartData = data.map((d) => ({
    x: d.x,
    value: d.y,
  }));

  const chartConfig = {
    value: {
      label: header.label,
      color,
    },
  } satisfies ChartConfig;

  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex gap-3 items-center">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          {header.icon}
        </div>
        <span className="text-gray-11 text-xs">{header.label}</span>
        <div className="ml-10">{header.value}</div>
      </div>
      <ChartContainer config={chartConfig} className="aspect-auto h-[48px] w-full">
        <BarChart data={chartData}>
          <ChartTooltip
            content={
              <ChartTooltipContent
                className="w-[150px]"
                labelFormatter={(value) =>
                  xAxisFormatter?.(value) ??
                  new Date(value).toLocaleDateString("en-US", {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })
                }
                formatter={(value) => tooltipFormatter?.(value as number) ?? value}
              />
            }
          />
          <Bar dataKey="value" fill={color} />
        </BarChart>
      </ChartContainer>
    </div>
  );
}
