"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { type InstanceRow, spansDays } from "@/lib/metrics/app-metrics";
import { IconChevronLeftOutline12, IconChevronRightOutline12 } from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import type { Route } from "next";
import { useState } from "react";
import { withEpisodeMarkers } from "./event-markers";
import { InstanceFilterBar } from "./instance-filters";
import { InstanceTable } from "./instance-table";
import type { FilterKey, InstancesTabView, LogsTarget } from "./instances-model";
import { UsageCardView } from "./usage-card";

const PAGE_SIZE = 10;

export type LayoutHandlers = {
  onOpen: (row: InstanceRow) => void;
  logsHref: (target: LogsTarget) => Route;
  onFilterChange: (key: FilterKey, values: string[]) => void;
};

type Props = LayoutHandlers & {
  view: Extract<InstancesTabView, { status: "ready" }>;
  markers: ChartMarkers;
  syncId: string;
};

export function InstancesLayout({
  view,
  markers,
  syncId,
  onOpen,
  logsHref,
  onFilterChange,
}: Props) {
  const [page, setPage] = useState(0);
  const [hoveredKey, setHoveredKey] = useState<string | null>(null);
  const all = view.rows;
  const pages = Math.max(1, Math.ceil(all.length / PAGE_SIZE));
  const current = Math.min(page, pages - 1);
  const first = current * PAGE_SIZE;
  const rows = all.slice(first, first + PAGE_SIZE);
  const chartMarkers = withEpisodeMarkers(markers, view.episodes, {
    logsHref,
    withDate: spansDays(view.grid),
  });
  return (
    <Card className="overflow-hidden">
      <div className="relative z-20 grid divide-y divide-grayA-4 md:grid-cols-2 md:divide-x md:divide-y-0">
        {view.usage.map((card) => (
          <UsageCardView
            key={card.metric}
            card={card}
            legend={view.legend}
            focusKey={hoveredKey}
            markers={chartMarkers}
            syncId={syncId}
          />
        ))}
      </div>
      {view.crashEventsFailed && (
        <p className="border-t border-grayA-4 px-4 py-2 text-xs text-gray-10">
          Crash events unavailable
        </p>
      )}
      <InstanceFilterBar filters={view.filters} onChange={onFilterChange} />
      <InstanceTable
        rows={rows}
        logsHref={logsHref}
        onOpen={onOpen}
        onHover={setHoveredKey}
        footer={
          <div className="flex items-center gap-3 border-t border-grayA-4 px-4 py-2.5 text-xs text-gray-10">
            <span>
              {first + 1}–{first + rows.length} of {all.length} instances
            </span>
            <span className="ml-auto tabular-nums">
              {current + 1} of {pages}
            </span>
            <span className="flex items-center gap-1">
              <Button
                variant="outline"
                size="icon"
                aria-label="Previous page"
                disabled={current === 0}
                onClick={() => setPage(current - 1)}
              >
                <IconChevronLeftOutline12 className="size-3" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                aria-label="Next page"
                disabled={current >= pages - 1}
                onClick={() => setPage(current + 1)}
              >
                <IconChevronRightOutline12 className="size-3" />
              </Button>
            </span>
          </div>
        }
      />
    </Card>
  );
}
