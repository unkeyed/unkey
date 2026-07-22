import { useMemo } from "react";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import type { VerificationBucket } from "./schema/analytics.schema";

type ChartRow = VerificationBucket & { label: string };

// The API buckets by hour for windows up to ~4 days and by day beyond, so the
// axis/tooltip label follows the same threshold.
const HOURLY_MAX_DAYS = 4;

const VALID_COLOR = "hsl(var(--gray-8))";
const ERROR_COLOR = "hsl(var(--error-9))";

type Props = {
  buckets: VerificationBucket[];
  /** Window length in days; selects hourly vs daily label formatting. */
  days: number;
};

/**
 * Format a bucket timestamp for the x-axis and tooltip. Short windows use an
 * hourly label; multi-day windows use a date label, matching the bucket
 * granularity the API returns for each range.
 */
function formatBucketTime(time: number, days: number): string {
  const date = new Date(time);
  if (days <= HOURLY_MAX_DAYS) {
    return date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  }
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Stacked bar chart of verifications over time: valid (gray) stacked with error
 * (red), one bar per API bucket. Self-contained — axes, grid, and a custom
 * tooltip — rather than pulling the dashboard's selection-enabled chart infra,
 * since the portal only needs a read-only view.
 */
export function VerificationsChart({ buckets, days }: Props) {
  const data = useMemo(
    () => buckets.map((b) => ({ ...b, label: formatBucketTime(b.time, days) })),
    [buckets, days],
  );

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barCategoryGap={2}>
        <CartesianGrid
          horizontal
          vertical={false}
          strokeDasharray="3 3"
          stroke="hsl(var(--gray-6))"
          strokeOpacity={0.5}
        />
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          minTickGap={24}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 11 }}
        />
        <YAxis
          width={40}
          tickLine={false}
          axisLine={false}
          allowDecimals={false}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 11 }}
        />
        <Tooltip cursor={{ fill: "hsl(var(--gray-3))" }} content={<ChartTooltip />} />
        <Bar dataKey="valid" stackId="v" fill={VALID_COLOR} radius={[0, 0, 0, 0]} />
        <Bar dataKey="error" stackId="v" fill={ERROR_COLOR} radius={[2, 2, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

/**
 * Recharts injects `active` and `payload` into the tooltip content element at
 * runtime; they are typed loosely here (as the dashboard's chart does) and
 * narrowed before use.
 */
function ChartTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: Array<{ payload?: unknown }>;
}) {
  if (!active || !payload?.length) {
    return null;
  }
  const bucket = payload[0]?.payload as ChartRow | undefined;
  if (!bucket) {
    return null;
  }
  return (
    <div className="rounded-md border border-gray-6 bg-background px-3 py-2 text-xs shadow-md">
      <div className="mb-1 font-medium text-gray-12">{bucket.label}</div>
      <div className="flex flex-col gap-1">
        <TooltipRow label="Total" count={bucket.total} swatch={null} />
        <TooltipRow label="Valid" count={bucket.valid} swatch={VALID_COLOR} />
        <TooltipRow label="Errors" count={bucket.error} swatch={ERROR_COLOR} />
      </div>
    </div>
  );
}

function TooltipRow({
  label,
  count,
  swatch,
}: {
  label: string;
  count: number;
  swatch: string | null;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="flex items-center gap-2 text-gray-11">
        {swatch ? (
          <span
            className="size-1.5 rounded-[1px]"
            style={{ backgroundColor: swatch }}
            aria-hidden
          />
        ) : (
          <span className="size-1.5" aria-hidden />
        )}
        {label}
      </span>
      <span className="text-gray-12 tabular-nums">{count.toLocaleString()}</span>
    </div>
  );
}
