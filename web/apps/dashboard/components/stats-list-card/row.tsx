"use client";

import { ListTableRow, listTable } from "@/components/list-views";
import { formatNumber } from "@/lib/fmt";
import {
  ChartWell,
  type StatsChartState,
  type StatsListCardLabels,
  type StatsListCardProps,
  statsSummaryOf,
} from ".";

const COUNTED: Record<StatsChartState["type"], boolean> = {
  loading: false,
  error: false,
  empty: true,
  data: true,
};

export type StatsListTable = ReturnType<typeof statsListTable>;

export function statsListTable({
  name,
  meta,
  labels,
}: {
  name: string;
  meta: string;
  labels: StatsListCardLabels;
}) {
  return listTable({
    minWidth: 720,
    columns: [
      { id: "name", header: name, width: "minmax(0,1.6fr)", skeleton: "title" },
      { id: "activity", header: "Activity", width: "minmax(0,1.4fr)", skeleton: "text" },
      { id: "success", header: labels.success, width: "minmax(0,0.6fr)", skeleton: "text" },
      { id: "error", header: labels.error, width: "minmax(0,0.6fr)", skeleton: "text" },
      { id: "meta", header: meta, width: "minmax(0,0.9fr)", skeleton: "text" },
    ],
  });
}

export function StatsListRow({
  table,
  href,
  ariaLabel,
  title,
  subtitle,
  buckets,
  isLoading,
  isError,
  labels,
  footerLeft,
}: StatsListCardProps & { table: StatsListTable }) {
  const { success, error, chart } = statsSummaryOf({ buckets, isLoading, isError });
  const countOf = (count: number) => (COUNTED[chart.type] ? formatNumber(count) : "—");

  return (
    <ListTableRow
      table={table}
      href={href}
      label={ariaLabel}
      cells={{
        name: (
          <span className="flex min-w-0 flex-col gap-1">
            <span className="truncate text-sm font-medium">{title}</span>
            {subtitle ? <span className="truncate font-mono text-gray-11">{subtitle}</span> : null}
          </span>
        ),
        activity: <ChartWell chart={chart} labels={labels} className="pointer-events-none h-7" />,
        success: <span className="tabular-nums text-gray-11">{countOf(success)}</span>,
        error: <span className="tabular-nums text-gray-11">{countOf(error)}</span>,
        meta: <span className="min-w-0 text-gray-11">{footerLeft}</span>,
      }}
    />
  );
}
