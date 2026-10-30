"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { type ChartState, METRIC_VIEWS } from "@/lib/metrics/views";
import { IconArrowsExpandDiagonalOutline18 } from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import { cn } from "cn";
import { absoluteScale } from "./allocation";
import type { MetricCardModel } from "./model";
import { SeriesChart } from "./series-chart";

const EXPANDABLE: Record<ChartState["status"], boolean> = {
  loading: true,
  failed: true,
  empty: false,
  ready: true,
};

type CardProps = {
  card: MetricCardModel;
  markers: ChartMarkers;
  height: number;
  syncId: string;
  onExpand: () => void;
};

export function MetricCard({ card, markers, height, syncId, onExpand }: CardProps) {
  const view = METRIC_VIEWS[card.metric];
  return (
    <Card className="group/card flex flex-col gap-1.5 px-4 pt-3 pb-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium text-gray-12">{view.title}</h3>
        {EXPANDABLE[card.state.status] && (
          <Button
            variant="ghost"
            size="icon"
            className="-mr-1.5 opacity-0 transition-opacity group-hover/card:opacity-100 focus-visible:opacity-100"
            aria-label={`Expand ${view.title}`}
            onClick={onExpand}
          >
            <IconArrowsExpandDiagonalOutline18 className="size-3 text-gray-10" />
          </Button>
        )}
      </div>
      <SeriesChart
        title={view.title}
        state={card.state}
        series={view.series}
        scale={absoluteScale(view.unit)}
        markers={markers}
        height={height}
        syncId={syncId}
        connectNulls={view.gaps === "bridge"}
        className={cn(
          "transition-opacity duration-150 ease-out",
          card.pending && "opacity-35 pointer-events-none",
        )}
      />
    </Card>
  );
}
