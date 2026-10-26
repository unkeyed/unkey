"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import type { InstanceRow } from "@/lib/metrics/app-metrics";
import { IconChevronLeftOutline12, IconChevronRightOutline12 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { Route } from "next";
import { useState } from "react";
import { InstanceTable } from "./instance-table";
import type { InstancesTabView } from "./instances-model";
import { UsageCardView } from "./usage-card";

const PAGE_SIZE = 10;

export type LayoutHandlers = {
  onOpen: (row: InstanceRow) => void;
  logsHref: (row: InstanceRow) => Route;
  onClearInstance: () => void;
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
  onClearInstance,
}: Props) {
  const [page, setPage] = useState(0);
  const all = view.rows;
  const pages = Math.max(1, Math.ceil(all.length / PAGE_SIZE));
  const current = Math.min(page, pages - 1);
  const first = current * PAGE_SIZE;
  const rows = all.slice(first, first + PAGE_SIZE);
  return (
    <>
      <div className="grid gap-4 md:grid-cols-2">
        {view.usage.map((card) => (
          <UsageCardView
            key={card.metric}
            card={card}
            legend={view.legend}
            selected={view.selected}
            markers={markers}
            syncId={syncId}
            onClearInstance={onClearInstance}
          />
        ))}
      </div>
      <InstanceTable
        rows={rows}
        logsHref={logsHref}
        onOpen={onOpen}
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
    </>
  );
}
