"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { type ChartIncidents, INCIDENT_FILL } from "@/components/charts/incident-markers";
import { METRIC_VIEWS } from "@/lib/metrics/views";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import type { ReactNode } from "react";
import { DeployIdentity } from "./deploy-identity";
import {
  type DeployRef,
  LINE_COLOR,
  type LineTone,
  type UsageCard,
  type UsageLegend,
} from "./instances-model";
import { LegendCell, SeriesChart } from "./series-chart";

const USAGE_HEIGHT = 180;

type LegendKey = {
  key: LineTone | "crashes";
  color: string;
  show: (legend: UsageLegend) => boolean;
  label: (legend: UsageLegend) => ReactNode;
};

const LEGEND_KEYS: LegendKey[] = [
  {
    key: "current",
    color: LINE_COLOR.current,
    show: (legend) => legend.current.length > 0,
    label: (legend) => <DeployKey label="Current" deploys={legend.current} />,
  },
  {
    key: "previous",
    color: LINE_COLOR.previous,
    show: (legend) => legend.previous.length > 0,
    label: (legend) => <DeployKey label="Previous" deploys={legend.previous} />,
  },
  {
    key: "unhealthy",
    color: LINE_COLOR.unhealthy,
    show: (legend) => legend.unhealthy,
    label: () => "Unhealthy",
  },
  {
    key: "stopped",
    color: LINE_COLOR.stopped,
    show: (legend) => legend.stopped,
    label: () => "Stopped",
  },
  {
    key: "crashes",
    color: INCIDENT_FILL,
    show: (legend) => legend.crashes > 0,
    label: (legend) => (
      <span className="flex items-baseline gap-1.5">
        Crashes
        <span className="tabular-nums text-gray-10">{legend.crashes}</span>
      </span>
    ),
  },
];

export function UsageCardView({
  card,
  legend,
  focusKey,
  markers,
  incidents,
  syncId,
}: {
  card: UsageCard;
  legend: UsageLegend;
  focusKey: string | null;
  markers: ChartMarkers;
  incidents: ChartIncidents;
  syncId: string;
}) {
  const view = METRIC_VIEWS[card.metric];
  return (
    <div className="flex min-w-0 flex-col gap-1.5 px-4 pt-3 pb-3">
      <h3 className="text-sm font-medium text-gray-12">
        {view.title}
        {card.allocation && <span className="font-normal text-gray-10"> · {card.allocation}</span>}
      </h3>
      <SeriesChart
        title={view.title}
        state={card.chart}
        series={card.series}
        scale={card.scale}
        markers={markers}
        incidents={incidents}
        height={USAGE_HEIGHT}
        syncId={syncId}
        legend={<Legend legend={legend} />}
        focusKey={focusKey}
        unavailableCopy="Usage is unavailable right now"
      />
    </div>
  );
}

function Legend({ legend }: { legend: UsageLegend }) {
  return LEGEND_KEYS.filter(({ show }) => show(legend)).map(({ key, color, label }) => (
    <LegendCell key={key} color={color} label={label(legend)} />
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
