"use client";

import type { AreaChartPoint } from "@/components/charts/area-timeseries";
import type { ChartMarkers } from "@/components/charts/chart-markers";
import { type ChartState, type MetricUnit, formatValue } from "@/lib/metrics/views";
import { IconArrowsExpandDiagonalOutline18 } from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import type { CardView, MetricCardModel } from "./model";
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
  const { view } = card;
  return (
    <Card className="group/card flex flex-col gap-1.5 px-4 pt-3 pb-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-medium text-gray-12">
          {view.title}
          {view.subtitle && <span className="font-normal text-gray-10"> · {view.subtitle}</span>}
        </h3>
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
        scale={view.scale}
        markers={markers}
        height={height}
        syncId={syncId}
        connectNulls={view.connectNulls}
        tooltipFooter={totalFooter(view)}
      />
    </Card>
  );
}

export function totalFooter(view: Pick<CardView, "total">) {
  const unit = view.total;
  return unit === null
    ? undefined
    : (point: AreaChartPoint) => <TotalRow unit={unit} value={point.total} />;
}

function TotalRow({ unit, value }: { unit: MetricUnit; value: number | undefined }) {
  if (value === undefined) {
    return null;
  }
  return (
    <div className="flex items-center gap-2 border-t border-grayA-4 pt-1.5 text-gray-11">
      <span>Total across instances</span>
      <span className="ml-auto font-mono tabular-nums text-gray-12">
        {formatValue(unit, value)}
      </span>
    </div>
  );
}
