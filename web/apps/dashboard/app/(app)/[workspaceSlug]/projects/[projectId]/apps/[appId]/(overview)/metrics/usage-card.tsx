"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { METRIC_VIEWS } from "@/lib/metrics/views";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { Card, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import type { ReactNode } from "react";
import { DeployIdentity } from "./deploy-identity";
import {
  type DeployRef,
  type InstanceView,
  LINE_COLOR,
  type LineTone,
  type UsageCard,
  type UsageLegend,
} from "./instances-model";
import { LegendCell, SeriesChart } from "./series-chart";

const USAGE_HEIGHT = 180;

const DEPLOY_KEYS: { tone: Exclude<LineTone, "unhealthy">; label: string }[] = [
  { tone: "current", label: "Current" },
  { tone: "previous", label: "Previous" },
];

export function UsageCardView({
  card,
  legend,
  selected,
  focusKey,
  markers,
  syncId,
  onClearInstance,
}: {
  onClearInstance: () => void;
  card: UsageCard;
  legend: UsageLegend;
  selected: InstanceView | null;
  focusKey: string | null;
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
          legendClassName="-mx-4 -mb-3"
          focusKey={focusKey}
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
  const showing = selected && (
    <span className="text-2xs text-gray-9">
      Showing <span className="font-mono text-gray-12">{selected.key}</span> ·{" "}
      <button
        type="button"
        onClick={onClear}
        className="rounded-sm underline outline-none hover:text-gray-12 focus-visible:ring-1 focus-visible:ring-grayA-12"
      >
        clear
      </button>
    </span>
  );
  const keys: { key: string; color: string; label: ReactNode }[] = [
    ...DEPLOY_KEYS.filter(({ tone }) => legend[tone].length > 0).map(({ tone, label }) => ({
      key: tone,
      color: LINE_COLOR[tone],
      label: <DeployKey label={label} deploys={legend[tone]} />,
    })),
    ...(legend.unhealthy
      ? [{ key: "unhealthy", color: LINE_COLOR.unhealthy, label: "Unhealthy" }]
      : []),
  ];
  return keys.map(({ key, color, label }, i) => (
    <LegendCell key={key} color={color} label={label}>
      {i === keys.length - 1 && showing}
    </LegendCell>
  ));
}

function DeployKey({ label, deploys }: { label: string; deploys: DeployRef[] }) {
  return (
    <span className="flex items-center gap-1.5">
      {label}
      <Popover>
        <PopoverTrigger
          openOnHover
          delay={0}
          closeDelay={100}
          aria-label={`${label} deployments`}
          className="-m-1 inline-flex cursor-default items-center rounded-sm p-1 text-gray-9 outline-none hover:text-gray-11 focus-visible:ring-1 focus-visible:ring-grayA-12"
        >
          <IconCircleInfoOutline18 className="size-3.5" aria-hidden="true" />
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
              className="min-w-0 gap-1"
            />
          ))}
        </PopoverContent>
      </Popover>
    </span>
  );
}
