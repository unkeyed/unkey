"use client";

import { type AreaChartPoint, AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import { RANGE_LABELS, UNITS, formatValue } from "@/lib/metrics/views";
import {
  IconChartActivityOutline18,
  IconGaugeOutline18,
  IconLink4Outline18,
  IconMicrochipOutline18,
} from "@unkey/icons";
import { SettingsGroup, SettingsGroupTitle } from "@unkey/ui";
import type { ComponentType } from "react";
import type { DatabaseVitals, MetricPoint } from "./database-model";
import { InfoCard, InfoCell, InfoCells } from "./info-card";

const SERIES = { queries: { label: "Queries", color: "var(--color-activity)" } };

type VitalId = "queries" | "latency" | "cpu" | "connections";

const VITALS: Record<
  VitalId,
  {
    label: string;
    icon: ComponentType<{ className?: string }>;
    format: (v: DatabaseVitals) => string;
  }
> = {
  queries: {
    label: "Queries/s",
    icon: IconChartActivityOutline18,
    format: (v) => UNITS.queriesPerSecond.format(queriesNow(v)).value,
  },
  latency: {
    label: "p95 latency",
    icon: IconGaugeOutline18,
    format: (v) => formatValue("milliseconds", v.latencyP95Ms),
  },
  cpu: {
    label: "CPU",
    icon: IconMicrochipOutline18,
    format: (v) => formatValue("ratio", v.cpuRatio),
  },
  connections: {
    label: "Connections",
    icon: IconLink4Outline18,
    format: (v) => formatValue("count", v.connections),
  },
};

function queriesNow(vitals: DatabaseVitals): number {
  return vitals.queries.at(-1)?.[1] ?? 0;
}

function chartPoints(points: readonly MetricPoint[]): AreaChartPoint[] {
  return points.map(([at, queries]) => ({ originalTimestamp: at * 1000, queries }));
}

function QueriesChart({ points }: { points: readonly MetricPoint[] }) {
  return (
    <AreaTimeseriesChart
      data={chartPoints(points)}
      config={SERIES}
      height={140}
      fill="flat"
      showZeroLine
      formatTooltipValue={UNITS.queriesPerSecond.format}
      axis={{ y: { tickCount: 3, formatTick: UNITS.queriesPerSecond.tick } }}
    />
  );
}

export function DatabaseActivity({ vitals }: { vitals: DatabaseVitals }) {
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Activity</SettingsGroupTitle>
      <InfoCard>
        <InfoCells columns={4}>
          {Object.entries(VITALS).map(([id, vital]) => (
            <InfoCell key={id} icon={vital.icon} label={vital.label} value={vital.format(vitals)} />
          ))}
        </InfoCells>
        <div className="flex flex-col gap-2 px-4 pt-3 pb-3">
          <div className="flex items-baseline justify-between gap-3">
            <h3 className="text-sm font-medium text-gray-12">Queries</h3>
            <span className="text-xs text-gray-10">{RANGE_LABELS["24h"]}</span>
          </div>
          <QueriesChart points={vitals.queries} />
        </div>
      </InfoCard>
    </SettingsGroup>
  );
}
