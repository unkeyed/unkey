import { useMemo } from "react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "~/components/ui/chart";
import { formatBucketTime, formatCount } from "./format";
import type { VerificationBucket } from "./schema/analytics.schema";

// Matches the dashboard's valid bars (`--accent-4`); the portal has no accent scale.
const VALID_BAR_COLOR = "hsl(240 10% 92%)";
export const VALID_COLOR = "hsl(var(--gray-8))";
export const REJECTED_COLOR = "hsl(var(--error-9))";
const GRID_COLOR = "hsl(var(--gray-6))";

const chartConfig: ChartConfig = {
  valid: { label: "Valid", color: VALID_BAR_COLOR },
  rejected: { label: "Invalid", color: REJECTED_COLOR },
};

type Row = {
  time: number;
  valid: number;
  rejected: number;
  total: number;
};

type Props = {
  buckets: VerificationBucket[];
  days: number;
};

export function VerificationsChart({ buckets, days }: Props) {
  const rows = useMemo<Row[]>(
    () =>
      buckets.map((bucket) => ({
        time: bucket.time,
        valid: bucket.valid,
        rejected: bucket.error,
        total: bucket.total,
      })),
    [buckets],
  );

  return (
    <ChartContainer config={chartConfig} className="aspect-auto h-full w-full">
      <BarChart
        accessibilityLayer
        data={rows}
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
          cursor={{ fill: "hsl(var(--gray-3))" }}
          content={
            <ChartTooltipContent
              labelFormatter={(_, payload) => {
                const row: Row | undefined = payload[0]?.payload;
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
        <Bar dataKey="valid" stackId="v" fill="var(--color-valid)" isAnimationActive={false} />
        <Bar
          dataKey="rejected"
          stackId="v"
          fill="var(--color-rejected)"
          radius={[2, 2, 0, 0]}
          isAnimationActive={false}
        />
      </BarChart>
    </ChartContainer>
  );
}
