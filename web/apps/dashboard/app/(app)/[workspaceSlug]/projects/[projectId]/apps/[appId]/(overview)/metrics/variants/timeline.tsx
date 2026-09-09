"use client";

import type { AppResourceMetric } from "@unkey/clickhouse";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ReactNode, useState } from "react";
import { EnvironmentSelect, Legend, WindowPills } from "../components/controls";
import { DeploymentPicker } from "../components/deployment-picker";
import { type DeployMarker, MetricsChart } from "../components/metrics-chart";
import {
  type DeploymentInfo,
  type MetricsScope,
  useEnvironmentDeployments,
  useLatencySeries,
  type useMetricsUrlState,
  useRequestSeries,
  useResourceSeries,
} from "../hooks/use-app-metrics";
import { useMetricsBundle } from "../hooks/use-metrics-bundle";
import {
  METRIC_COLORS,
  PERCENTILE_SERIES,
  STATUS_SERIES,
  type SeriesKey,
  type SeriesPoint,
  formatBytes,
  formatBytesRate,
  formatMs,
  formatPercent,
  formatRequestRate,
  formatVcpu,
  paletteColor,
  pivot,
  seriesKeysOf,
  shortSha,
  tickBytes,
  tickBytesRate,
  tickCount,
  tickMs,
  tickPercent,
  tickVcpu,
} from "../lib/series";

type UrlState = ReturnType<typeof useMetricsUrlState>;

// Which deployments the charts are split by, with a stable colour each.
type Selection = {
  ids: string[];
  series: SeriesKey[];
  colorOf: (id: string) => string;
};

// Timeline direction: a grid of equal charts, one metric each, every chart on
// the same time axis with dotted deploy lines. Resource cards toggle between
// the summed line and one line per instance. Picking deployments at the top
// turns every chart into one line per deployment instead.
export function TimelineVariant({ state }: { state: UrlState & { environmentId: string } }) {
  const deployments = useEnvironmentDeployments(state.environmentId);
  const colorOf = (id: string) =>
    paletteColor(
      Math.max(
        0,
        deployments.findIndex((d) => d.id === id),
      ),
    );
  const ids = state.selectedDeployments.filter((id) => deployments.some((d) => d.id === id));
  const selection: Selection = {
    ids,
    colorOf,
    series: ids.map((id) => {
      const d = deployments.find((x) => x.id === id);
      return { key: id, label: shortSha(d?.sha) || id.slice(-7), color: colorOf(id) };
    }),
  };
  const split = ids.length > 0;

  const scope: MetricsScope = {
    appId: state.appId,
    environmentId: state.environmentId,
    window: state.window,
    groupBy: split ? "deployment" : "none",
  };
  const bundle = useMetricsBundle(scope);
  const domain: [number, number] = [bundle.range.startMs, bundle.range.endMs];
  const markers = split ? bundle.markers.filter((m) => ids.includes(m.id)) : bundle.markers;
  const chart: ChartShared = {
    domain,
    bucketSeconds: bundle.range.bucketSeconds,
    markers,
    showMarkerLabels: false,
    syncId: "app-metrics",
  };
  const inRange = (d: DeploymentInfo) => d.createdAt >= domain[0] && d.createdAt < domain[1];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <EnvironmentSelect
            environments={state.environments}
            value={state.environmentId}
            onChange={state.setEnvironmentId}
          />
          <DeploymentPicker
            deployments={deployments}
            colorOf={colorOf}
            selected={ids}
            onChange={state.setSelectedDeployments}
            inRange={inRange}
          />
        </div>
        <WindowPills value={state.window} onChange={state.setWindow} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <ResourceCard title="CPU" metric="cpu" scope={scope} chart={chart} selection={selection} />
        <ResourceCard
          title="Memory"
          metric="memory"
          scope={scope}
          chart={chart}
          selection={selection}
        />
        <NetworkCard bundle={bundle} chart={chart} selection={selection} />
        <RequestsCard scope={scope} chart={chart} selection={selection} />
        <ErrorRateCard scope={scope} chart={chart} selection={selection} />
        <LatencyCard scope={scope} chart={chart} selection={selection} />
        <ResourceCard
          title="Disk"
          metric="disk"
          scope={scope}
          chart={chart}
          selection={selection}
        />
      </div>
    </div>
  );
}

type ChartShared = {
  domain: [number, number];
  bucketSeconds: number;
  markers: DeployMarker[];
  showMarkerLabels: boolean;
  syncId: string;
};

function Card({
  title,
  right,
  children,
}: {
  title: ReactNode;
  right?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="border border-gray-4 bg-grayA-1 rounded-lg flex flex-col min-w-0">
      <header className="flex items-center justify-between gap-3 px-4 pt-3 pb-1">
        <h3 className="text-[13px] font-medium text-gray-12">{title}</h3>
        {right}
      </header>
      <div className="px-2 pb-2">{children}</div>
    </section>
  );
}

function SplitToggle({
  value,
  onChange,
  labels,
}: {
  value: boolean;
  onChange: (v: boolean) => void;
  labels: [string, string];
}) {
  return (
    <div role="radiogroup" className="flex items-center gap-3 text-[11px]">
      {([false, true] as const).map((split, i) => (
        <button
          key={labels[i]}
          type="button"
          role="radio"
          aria-checked={value === split}
          onClick={() => onChange(split)}
          className={cn(
            "flex items-center gap-1.5 transition-colors",
            value === split ? "text-gray-12" : "text-gray-9 hover:text-gray-11",
          )}
        >
          <span
            className={cn(
              "size-2 rounded-full border",
              value === split ? "bg-accent-9 border-accent-9" : "border-gray-8",
            )}
          />
          {labels[i]}
        </button>
      ))}
    </div>
  );
}

const RESOURCE_FORMAT: Record<
  AppResourceMetric,
  {
    scale: (y: number, bucketSeconds: number) => number;
    format: (v: number) => string;
    tick: (v: number) => string;
  }
> = {
  cpu: { scale: (y, b) => y / b / 1000, format: formatVcpu, tick: tickVcpu },
  memory: { scale: (y) => y, format: (v) => formatBytes(v), tick: tickBytes },
  disk: { scale: (y) => y, format: (v) => formatBytes(v), tick: tickBytes },
  egress: { scale: (y, b) => y / b, format: formatBytesRate, tick: tickBytesRate },
  ingress: { scale: (y, b) => y / b, format: formatBytesRate, tick: tickBytesRate },
};

function ResourceCard({
  title,
  metric,
  scope,
  chart,
  selection,
}: {
  title: string;
  metric: AppResourceMetric;
  scope: MetricsScope;
  chart: ChartShared;
  selection: Selection;
}) {
  const [perInstance, setPerInstance] = useState(false);
  const byDeployment = selection.ids.length > 0;
  const query = useResourceSeries(
    { ...scope, groupBy: byDeployment ? "deployment" : perInstance ? "instance" : "none" },
    metric,
  );
  const rows = query.data?.points ?? [];
  const range = query.data?.range ?? {
    startMs: chart.domain[0],
    endMs: chart.domain[1],
    bucketSeconds: chart.bucketSeconds,
  };
  const series: SeriesKey[] = byDeployment
    ? selection.series
    : perInstance
      ? seriesKeysOf(rows).map((k, i) => ({ key: k, label: k, color: paletteColor(i) }))
      : [{ key: "sum", label: "Sum", color: METRIC_COLORS[metric] }];
  const { scale, format, tick } = RESOURCE_FORMAT[metric];
  const points = pivot(
    rows.map((r) => ({ ...r, series: r.series || "sum" })),
    range,
    series.map((s) => s.key),
    (y) => scale(y, range.bucketSeconds),
  );

  return (
    <Card
      title={title}
      right={
        byDeployment ? (
          <Legend items={series} />
        ) : (
          <SplitToggle
            value={perInstance}
            onChange={setPerInstance}
            labels={["Sum", "Instances"]}
          />
        )
      }
    >
      <MetricsChart
        points={points}
        series={series}
        kind={byDeployment || perInstance ? "line" : "area"}
        {...chart}
        formatValue={format}
        formatTick={tick}
        isLoading={query.isLoading}
        isError={query.isError}
      />
    </Card>
  );
}

function NetworkCard({
  bundle,
  chart,
  selection,
}: {
  bundle: ReturnType<typeof useMetricsBundle>;
  chart: ChartShared;
  selection: Selection;
}) {
  const b = bundle.range.bucketSeconds;
  const byDeployment = selection.ids.length > 0;

  if (byDeployment) {
    const points = pivot(
      bundle.egress.data?.points ?? [],
      bundle.range,
      selection.ids,
      (y) => y / b,
    );
    return (
      <Card
        title={
          <>
            Public Network Traffic <span className="text-gray-10 font-normal">egress</span>
          </>
        }
        right={<Legend items={selection.series} />}
      >
        <MetricsChart
          points={points}
          series={selection.series}
          kind="line"
          {...chart}
          formatValue={formatBytesRate}
          formatTick={tickBytesRate}
          isLoading={bundle.egress.isLoading}
          isError={bundle.egress.isError}
        />
      </Card>
    );
  }

  const series: SeriesKey[] = [
    { key: "egress", label: "Egress", color: METRIC_COLORS.egress },
    { key: "ingress", label: "Ingress", color: METRIC_COLORS.ingress },
  ];
  const egress = pivot(
    (bundle.egress.data?.points ?? []).map((r) => ({ ...r, series: "egress" })),
    bundle.range,
    ["egress"],
    (y) => y / b,
  );
  const ingress = pivot(
    (bundle.ingress.data?.points ?? []).map((r) => ({ ...r, series: "ingress" })),
    bundle.range,
    ["ingress"],
    (y) => y / b,
  );
  const points = egress.map((p, i) => ({ ...p, ingress: ingress[i]?.ingress ?? 0 }));

  return (
    <Card title="Public Network Traffic" right={<Legend items={series} />}>
      <MetricsChart
        points={points}
        series={series}
        kind="area"
        {...chart}
        formatValue={formatBytesRate}
        formatTick={tickBytesRate}
        isLoading={bundle.egress.isLoading || bundle.ingress.isLoading}
        isError={bundle.egress.isError || bundle.ingress.isError}
      />
    </Card>
  );
}

// Request rows arrive as "deployment:2xx" when grouped by deployment. Folds
// them into one row per bucket with `<id>` (total) and `<id>:5xx` columns.
function requestsByDeployment(
  rows: { x: number; series: string; y: number }[],
  range: { startMs: number; endMs: number; bucketSeconds: number },
  ids: string[],
): SeriesPoint[] {
  const keys = ids.flatMap((id) => [id, `${id}:5xx`]);
  const points = pivot([], range, keys);
  const byX = new Map(points.map((p) => [p.x, p]));
  for (const r of rows) {
    const [id, klass] = r.series.split(":");
    const row = byX.get(r.x);
    if (!row || !ids.includes(id)) {
      continue;
    }
    row[id] = (row[id] ?? 0) + r.y;
    if (klass === "5xx") {
      row[`${id}:5xx`] = (row[`${id}:5xx`] ?? 0) + r.y;
    }
  }
  return points;
}

function RequestsCard({
  scope,
  chart,
  selection,
}: {
  scope: MetricsScope;
  chart: ChartShared;
  selection: Selection;
}) {
  const query = useRequestSeries(scope);
  const range = query.data?.range;
  const b = range?.bucketSeconds ?? chart.bucketSeconds;
  const byDeployment = selection.ids.length > 0;

  if (byDeployment) {
    const points = range
      ? requestsByDeployment(query.data?.points ?? [], range, selection.ids).map((p) => {
          const out: SeriesPoint = { x: p.x };
          for (const id of selection.ids) {
            out[id] = (p[id] ?? 0) / b;
          }
          return out;
        })
      : [];
    return (
      <Card title="Requests" right={<Legend items={selection.series} />}>
        <MetricsChart
          points={points}
          series={selection.series}
          kind="line"
          {...chart}
          formatValue={formatRequestRate}
          formatTick={(v) => (v > 0 ? `${v < 10 ? v.toFixed(1) : tickCount(v)}/s` : "")}
          isLoading={query.isLoading}
          isError={query.isError}
          emptyMessage="No requests in this range"
        />
      </Card>
    );
  }

  const points = range
    ? pivot(
        query.data?.points ?? [],
        range,
        STATUS_SERIES.map((s) => s.key),
        (y) => y / b,
      )
    : [];
  return (
    <Card title="Requests" right={<Legend items={STATUS_SERIES} />}>
      <MetricsChart
        points={points}
        series={STATUS_SERIES}
        kind="area"
        stacked
        {...chart}
        formatValue={formatRequestRate}
        formatTick={(v) => (v > 0 ? `${v < 10 ? v.toFixed(1) : tickCount(v)}/s` : "")}
        isLoading={query.isLoading}
        isError={query.isError}
        emptyMessage="No requests in this range"
      />
    </Card>
  );
}

function ErrorRateCard({
  scope,
  chart,
  selection,
}: {
  scope: MetricsScope;
  chart: ChartShared;
  selection: Selection;
}) {
  const query = useRequestSeries(scope);
  const range = query.data?.range;
  const byDeployment = selection.ids.length > 0;

  let points: SeriesPoint[] = [];
  let series: SeriesKey[];
  if (byDeployment) {
    series = selection.series;
    points = range
      ? requestsByDeployment(query.data?.points ?? [], range, selection.ids).map((p) => {
          const out: SeriesPoint = { x: p.x };
          for (const id of selection.ids) {
            const total = p[id] ?? 0;
            out[id] = total > 0 ? ((p[`${id}:5xx`] ?? 0) / total) * 100 : 0;
          }
          return out;
        })
      : [];
  } else {
    series = [{ key: "rate", label: "5xx rate", color: METRIC_COLORS.errors }];
    const counts = range
      ? pivot(
          query.data?.points ?? [],
          range,
          STATUS_SERIES.map((s) => s.key),
        )
      : [];
    points = counts.map((p) => {
      const total = STATUS_SERIES.reduce((acc, s) => acc + (p[s.key] ?? 0), 0);
      return { x: p.x, rate: total > 0 ? ((p["5xx"] ?? 0) / total) * 100 : 0 };
    });
  }
  const maxRate = points.reduce((m, p) => Math.max(m, ...series.map((s) => p[s.key] ?? 0)), 0);
  return (
    <Card title="Request Error Rate" right={<Legend items={series} />}>
      <MetricsChart
        points={points}
        series={series}
        kind="line"
        {...chart}
        formatValue={formatPercent}
        formatTick={tickPercent}
        yMax={Math.max(1, maxRate) * 1.15}
        isLoading={query.isLoading}
        isError={query.isError}
        emptyMessage="No errors in this range"
      />
    </Card>
  );
}

function LatencyCard({
  scope,
  chart,
  selection,
}: {
  scope: MetricsScope;
  chart: ChartShared;
  selection: Selection;
}) {
  const query = useLatencySeries(scope);
  const rows = query.data?.points ?? [];
  const range = query.data?.range;
  const byDeployment = selection.ids.length > 0;

  if (byDeployment) {
    const points = range
      ? pivot(
          rows.map((r) => ({ x: r.x, series: r.series, y: r.p99 })),
          range,
          selection.ids,
        )
      : [];
    return (
      <Card
        title={
          <>
            Response Time <span className="text-gray-10 font-normal">p99</span>
          </>
        }
        right={<Legend items={selection.series} />}
      >
        <MetricsChart
          points={points}
          series={selection.series}
          kind="line"
          {...chart}
          formatValue={formatMs}
          formatTick={tickMs}
          isLoading={query.isLoading}
          isError={query.isError}
          emptyMessage="No requests in this range"
        />
      </Card>
    );
  }

  const byX = new Map(rows.map((r) => [r.x, r]));
  const points = range
    ? pivot([], range, ["p50", "p95", "p99"]).map((p) => {
        const r = byX.get(p.x);
        return { x: p.x, p50: r?.p50 ?? 0, p95: r?.p95 ?? 0, p99: r?.p99 ?? 0 };
      })
    : [];
  return (
    <Card title="Response Time" right={<Legend items={PERCENTILE_SERIES} />}>
      <MetricsChart
        points={points}
        series={PERCENTILE_SERIES}
        kind="line"
        {...chart}
        formatValue={formatMs}
        formatTick={tickMs}
        isLoading={query.isLoading}
        isError={query.isError}
        emptyMessage="No requests in this range"
      />
    </Card>
  );
}
