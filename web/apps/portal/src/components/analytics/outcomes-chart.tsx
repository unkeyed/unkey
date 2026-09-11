import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "~/components/ui/chart";
import { formatBucketTime, formatCount } from "./format";
import { CHART_FILLS, CHART_KINDS, CHART_LABELS } from "./outcomes";
import type { VerificationBucket } from "./schema/analytics.schema";

const GRID_COLOR = "hsl(var(--gray-6))";

const chartConfig: ChartConfig = Object.fromEntries(
  CHART_KINDS.map((kind) => [kind, { label: CHART_LABELS[kind], color: CHART_FILLS[kind] }]),
);

type Props = {
  buckets: VerificationBucket[];
  days: number;
};

/** Valid at the base, then every rejection outcome present in the range in its own colour. */
export function OutcomesChart({ buckets, days }: Props) {
  const present = CHART_KINDS.filter((kind) => buckets.some((b) => b[kind] > 0));

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-full w-full">
      <BarChart
        accessibilityLayer
        data={buckets}
        margin={{ top: 8, right: 8, bottom: 0, left: 0 }}
        barCategoryGap={2}
      >
        <CartesianGrid
          horizontal
          vertical={false}
          strokeDasharray="3 3"
          stroke={GRID_COLOR}
          strokeOpacity={0.5}
        />
        <XAxis
          dataKey="time"
          tickLine={false}
          axisLine={false}
          minTickGap={32}
          tickFormatter={(time: number) => formatBucketTime(time, days)}
          tick={{ fontSize: 11 }}
        />
        <YAxis hide />
        <ChartTooltip
          cursor={{
            fill: "hsl(var(--accent-3))",
            strokeWidth: 1,
            strokeDasharray: "5 5",
            strokeOpacity: 0.7,
          }}
          content={
            <ChartTooltipContent
              labelFormatter={(_, payload) => {
                const row: VerificationBucket | undefined = payload[0]?.payload;
                if (!row) {
                  return null;
                }
                return (
                  <div className="flex items-center justify-between gap-8 px-4">
                    <span className="text-gray-12">{formatBucketTime(row.time, days)}</span>
                    <span className="font-mono text-gray-11 tabular-nums">
                      {formatCount(row.total)} total
                    </span>
                  </div>
                );
              }}
            />
          }
        />
        {present.map((kind, i) => (
          <Bar
            key={kind}
            dataKey={kind}
            stackId="v"
            fill={`var(--color-${kind})`}
            radius={i === present.length - 1 ? [2, 2, 0, 0] : undefined}
            isAnimationActive={false}
          />
        ))}
      </BarChart>
    </ChartContainer>
  );
}
