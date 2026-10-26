"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { METRIC_VIEWS } from "@/lib/metrics/views";
import { Card, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { DeployIdentity } from "./deploy-identity";
import {
  type DeployRef,
  type InstanceView,
  LINE_COLOR,
  type LineTone,
  type UsageCard,
  type UsageLegend,
} from "./instances-model";
import { SeriesChart } from "./series-chart";

const USAGE_HEIGHT = 180;

const DEPLOY_KEYS: { tone: Exclude<LineTone, "unhealthy">; label: string }[] = [
  { tone: "current", label: "Current" },
  { tone: "previous", label: "Previous" },
];

export function UsageCardView({
  card,
  legend,
  selected,
  markers,
  syncId,
  onClearInstance,
}: {
  onClearInstance: () => void;
  card: UsageCard;
  legend: UsageLegend;
  selected: InstanceView | null;
  markers: ChartMarkers;
  syncId: string;
}) {
  const view = METRIC_VIEWS[card.metric];
  return (
    <Card className="relative z-20">
      <div className="flex flex-col gap-1.5 px-4 pt-3 pb-3">
        <h3 className="text-sm font-medium text-gray-12">
          {view.title}
          {card.allocation && (
            <span className="font-normal text-gray-10"> · {card.allocation}</span>
          )}
        </h3>
        <SeriesChart
          title={view.title}
          state={card.chart}
          series={card.series}
          scale={card.scale}
          markers={markers}
          height={USAGE_HEIGHT}
          syncId={syncId}
          legend={<Legend legend={legend} selected={selected} onClear={onClearInstance} />}
          focusKey={selected?.key ?? null}
          unavailableCopy="Usage is unavailable right now"
        />
      </div>
    </Card>
  );
}

function Legend({
  legend,
  selected,
  onClear,
}: {
  legend: UsageLegend;
  selected: InstanceView | null;
  onClear: () => void;
}) {
  return (
    <div className="flex min-h-4 flex-wrap items-center gap-x-3.5 gap-y-1 text-xs">
      {DEPLOY_KEYS.map(
        ({ tone, label }) =>
          legend[tone].length > 0 && (
            <DeployKey key={tone} color={LINE_COLOR[tone]} label={label} deploys={legend[tone]} />
          ),
      )}
      {legend.unhealthy && (
        <span className="flex items-center gap-1.5">
          <span className="size-2 rounded-full" style={{ backgroundColor: LINE_COLOR.unhealthy }} />
          <span className="text-gray-11">Unhealthy</span>
        </span>
      )}
      {selected && (
        <span className="ml-auto text-2xs text-gray-9">
          Showing <span className="font-mono text-gray-12">{selected.key}</span> ·{" "}
          <button
            type="button"
            onClick={onClear}
            className="rounded-sm underline outline-none hover:text-gray-12 focus-visible:ring-1 focus-visible:ring-grayA-12"
          >
            clear
          </button>
        </span>
      )}
    </div>
  );
}

function DeployKey({
  color,
  label,
  deploys,
}: { color: string; label: string; deploys: DeployRef[] }) {
  return (
    <Popover>
      <PopoverTrigger
        openOnHover
        delay={0}
        closeDelay={100}
        className="flex cursor-default items-center gap-1.5 rounded-sm outline-none focus-visible:ring-1 focus-visible:ring-grayA-12"
      >
        <span className="size-2 rounded-full" style={{ backgroundColor: color }} />
        <span className="text-gray-11 underline decoration-grayA-6 decoration-dotted underline-offset-4">
          {label}
        </span>
      </PopoverTrigger>
      <PopoverContent
        side="bottom"
        align="start"
        className="flex w-64 flex-col gap-2.5 px-3 py-2 text-2xs"
      >
        {deploys.map((deploy) => (
          <DeployIdentity
            key={deploy.id}
            source={deploy.source}
            deploymentId={deploy.id}
            aside={
              deploy.source && <span className="shrink-0 font-mono text-gray-10">{deploy.id}</span>
            }
            className="min-w-0 gap-1"
          />
        ))}
      </PopoverContent>
    </Popover>
  );
}
