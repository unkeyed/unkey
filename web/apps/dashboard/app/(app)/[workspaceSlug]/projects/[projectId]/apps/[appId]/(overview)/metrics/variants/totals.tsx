"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { ChevronLeft, ChevronRight } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import type { ReactNode } from "react";
import { useProjectData } from "../../data-provider";
import { EnvironmentSelect, WindowSelect } from "../components/controls";
import { DeploymentPicker } from "../components/deployment-picker";
import { MetricsChart } from "../components/metrics-chart";
import {
  type DeploymentInfo,
  type MetricsScope,
  useDeploymentIndex,
  useEnvironmentDeployments,
  type useMetricsUrlState,
} from "../hooks/use-app-metrics";
import { type MetricsBundle, useMetricsBundle } from "../hooks/use-metrics-bundle";
import {
  METRIC_COLORS,
  PERCENTILE_SERIES,
  STATUS_SERIES,
  type SeriesKey,
  type SeriesPoint,
  formatBytes,
  formatCount,
  formatCpuTime,
  formatMs,
  formatPercent,
  formatVcpu,
  maxSeries,
  meanSeries,
  paletteColor,
  pivot,
  seriesKeysOf,
  shortSha,
  sumSeries,
  tickBytes,
  tickCount,
  tickMs,
  tickPercent,
  tickVcpu,
} from "../lib/series";

type UrlState = ReturnType<typeof useMetricsUrlState>;

const DETAILS = ["requests", "transfer", "cpu", "memory", "latency", "errors", "disk"] as const;
type Detail = (typeof DETAILS)[number];

const DETAIL_TITLES: Record<Detail, string> = {
  requests: "Requests",
  transfer: "Data Transfer",
  cpu: "CPU",
  memory: "Memory",
  latency: "Response Time",
  errors: "Errors",
  disk: "Disk",
};

function isDetail(v: string | null): v is Detail {
  return v !== null && (DETAILS as readonly string[]).includes(v);
}

// Totals direction: an overview of cards that lead with totals, and a
// drill-in per metric that splits the chart by deployment and lists every
// deployment's share in a table. The question "which deploy did this" is
// answered by a table row, not by reading a marker off a line.
export function TotalsVariant({ state }: { state: UrlState & { environmentId: string } }) {
  const detail = isDetail(state.detail) ? state.detail : null;
  const deployments = useEnvironmentDeployments(state.environmentId);
  const colorOf = (id: string) =>
    paletteColor(
      Math.max(
        0,
        deployments.findIndex((d) => d.id === id),
      ),
    );
  const selected = state.selectedDeployments.filter((id) => deployments.some((d) => d.id === id));
  const scope: MetricsScope = {
    appId: state.appId,
    environmentId: state.environmentId,
    window: state.window,
    groupBy: detail || selected.length > 0 ? "deployment" : "none",
  };
  const rawBundle = useMetricsBundle(scope);
  const bundle: MetricsBundle =
    selected.length > 0
      ? { ...rawBundle, markers: rawBundle.markers.filter((m) => selected.includes(m.id)) }
      : rawBundle;
  const domain: [number, number] = [bundle.range.startMs, bundle.range.endMs];
  const inRange = (d: DeploymentInfo) => d.createdAt >= domain[0] && d.createdAt < domain[1];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          {detail && (
            <button
              type="button"
              onClick={() => state.setDetail(null)}
              className="inline-flex items-center gap-1 h-8 pl-1.5 pr-2.5 rounded-md text-[13px] text-gray-11 hover:text-gray-12 hover:bg-grayA-3 transition-colors"
            >
              <ChevronLeft iconSize="sm-regular" />
              Overview
            </button>
          )}
          <EnvironmentSelect
            environments={state.environments}
            value={state.environmentId}
            onChange={state.setEnvironmentId}
          />
          <DeploymentPicker
            deployments={deployments}
            colorOf={colorOf}
            selected={selected}
            onChange={state.setSelectedDeployments}
            inRange={inRange}
          />
        </div>
        <WindowSelect value={state.window} onChange={state.setWindow} />
      </div>

      {detail ? (
        <DetailView
          detail={detail}
          bundle={bundle}
          appId={state.appId}
          selected={selected}
          colorOf={colorOf}
        />
      ) : selected.length > 0 ? (
        <Overview
          bundle={bundle}
          onOpen={(d) => state.setDetail(d)}
          selected={selected}
          colorOf={colorOf}
        />
      ) : (
        <Overview
          bundle={bundle}
          onOpen={(d) => state.setDetail(d)}
          selected={[]}
          colorOf={colorOf}
        />
      )}
    </div>
  );
}

// ─── Overview ────────────────────────────────────────────────────────

type Shaped = {
  points: SeriesPoint[];
  series: SeriesKey[];
  stats: { label: string; value: string; color?: string }[];
  kind: "area" | "bar" | "line";
  stacked?: boolean;
  format: (v: number) => string;
  tick: (v: number) => string;
  isLoading: boolean;
  isError: boolean;
  yMax?: number;
};

// Turns the bundle into what each card draws. Overview totals use the
// unsplit series; totals per bucket (bars) for counters, levels (areas) for
// gauges.
function shape(bundle: MetricsBundle, detail: Detail): Shaped {
  const { range } = bundle;
  const b = range.bucketSeconds;
  const single = (
    q: MetricsBundle["cpu"],
    key: string,
    scale: (y: number) => number = (y) => y,
  ): SeriesPoint[] =>
    pivot(
      (q.data?.points ?? []).map((r) => ({ ...r, series: key })),
      range,
      [key],
      scale,
    );

  switch (detail) {
    case "requests": {
      const points = pivot(
        bundle.requests.data?.points ?? [],
        range,
        STATUS_SERIES.map((s) => s.key),
      );
      const total = STATUS_SERIES.reduce((acc, s) => acc + sumSeries(points, s.key), 0);
      const errors = sumSeries(points, "5xx");
      return {
        points,
        series: STATUS_SERIES,
        kind: "bar",
        stacked: true,
        format: formatCount,
        tick: tickCount,
        stats: [
          { label: "Total", value: formatCount(total) },
          { label: "5xx", value: formatPercent(total > 0 ? (errors / total) * 100 : 0) },
        ],
        isLoading: bundle.requests.isLoading,
        isError: bundle.requests.isError,
      };
    }
    case "transfer": {
      const egress = single(bundle.egress, "egress");
      const ingress = single(bundle.ingress, "ingress");
      const points = egress.map((p, i) => ({ ...p, ingress: ingress[i]?.ingress ?? 0 }));
      return {
        points,
        series: [
          { key: "egress", label: "Outgoing", color: METRIC_COLORS.egress },
          { key: "ingress", label: "Incoming", color: METRIC_COLORS.ingress },
        ],
        kind: "bar",
        format: (v) => formatBytes(v),
        tick: tickBytes,
        stats: [
          {
            label: "Outgoing",
            value: formatBytes(sumSeries(points, "egress")),
            color: METRIC_COLORS.egress,
          },
          {
            label: "Incoming",
            value: formatBytes(sumSeries(points, "ingress")),
            color: METRIC_COLORS.ingress,
          },
        ],
        isLoading: bundle.egress.isLoading || bundle.ingress.isLoading,
        isError: bundle.egress.isError || bundle.ingress.isError,
      };
    }
    case "cpu": {
      const points = single(bundle.cpu, "cpu", (y) => y / b / 1000);
      const cpuSeconds = (bundle.cpu.data?.points ?? []).reduce((acc, r) => acc + r.y / 1e6, 0);
      return {
        points,
        series: [{ key: "cpu", label: "CPU", color: METRIC_COLORS.cpu }],
        kind: "area",
        format: formatVcpu,
        tick: tickVcpu,
        stats: [
          { label: "CPU time", value: formatCpuTime(cpuSeconds) },
          { label: "Avg", value: formatVcpu(meanSeries(points, "cpu")) },
        ],
        isLoading: bundle.cpu.isLoading,
        isError: bundle.cpu.isError,
      };
    }
    case "memory": {
      const points = single(bundle.memory, "memory");
      return {
        points,
        series: [{ key: "memory", label: "Memory", color: METRIC_COLORS.memory }],
        kind: "area",
        format: (v) => formatBytes(v),
        tick: tickBytes,
        stats: [
          { label: "Peak", value: formatBytes(maxSeries(points, "memory")) },
          { label: "Avg", value: formatBytes(meanSeries(points, "memory")) },
        ],
        isLoading: bundle.memory.isLoading,
        isError: bundle.memory.isError,
      };
    }
    case "disk": {
      const points = single(bundle.disk, "disk");
      return {
        points,
        series: [{ key: "disk", label: "Disk", color: METRIC_COLORS.disk }],
        kind: "area",
        format: (v) => formatBytes(v),
        tick: tickBytes,
        stats: [{ label: "Peak used", value: formatBytes(maxSeries(points, "disk")) }],
        isLoading: bundle.disk.isLoading,
        isError: bundle.disk.isError,
      };
    }
    case "latency": {
      const byX = new Map((bundle.latency.data?.points ?? []).map((r) => [r.x, r]));
      const points = pivot([], range, ["p50", "p95", "p99"]).map((p) => {
        const r = byX.get(p.x);
        return { x: p.x, p50: r?.p50 ?? 0, p95: r?.p95 ?? 0, p99: r?.p99 ?? 0 };
      });
      return {
        points,
        series: PERCENTILE_SERIES,
        kind: "line",
        format: formatMs,
        tick: tickMs,
        stats: [
          { label: "p50", value: formatMs(meanSeries(points, "p50")) },
          { label: "p99", value: formatMs(meanSeries(points, "p99")) },
        ],
        isLoading: bundle.latency.isLoading,
        isError: bundle.latency.isError,
      };
    }
    case "errors": {
      const counts = pivot(
        bundle.requests.data?.points ?? [],
        range,
        STATUS_SERIES.map((s) => s.key),
      );
      const points = counts.map((p) => {
        const total = STATUS_SERIES.reduce((acc, s) => acc + (p[s.key] ?? 0), 0);
        return {
          x: p.x,
          rate: total > 0 ? ((p["5xx"] ?? 0) / total) * 100 : 0,
          count: p["5xx"] ?? 0,
        };
      });
      const total = STATUS_SERIES.reduce((acc, s) => acc + sumSeries(counts, s.key), 0);
      const errors = sumSeries(counts, "5xx");
      return {
        points,
        series: [{ key: "rate", label: "5xx rate", color: METRIC_COLORS.errors }],
        kind: "area",
        format: formatPercent,
        tick: tickPercent,
        yMax: Math.max(1, ...points.map((p) => p.rate)) * 1.15,
        stats: [
          { label: "5xx", value: formatCount(errors) },
          { label: "Rate", value: formatPercent(total > 0 ? (errors / total) * 100 : 0) },
        ],
        isLoading: bundle.requests.isLoading,
        isError: bundle.requests.isError,
      };
    }
  }
}

// Overview card content when deployments are picked: one series per pick,
// totals summed over the picks only.
function shapeSelected(
  bundle: MetricsBundle,
  detail: Detail,
  split: ReturnType<typeof splitByDeployment>,
  selected: string[],
  colorOf: (id: string) => string,
  index: Map<string, DeploymentInfo>,
): Shaped {
  const base = shape(bundle, detail);
  const b = bundle.range.bucketSeconds;
  const series: SeriesKey[] = selected.map((id) => ({
    key: id,
    label: shortSha(index.get(id)?.sha) || id.slice(-7),
    color: colorOf(id),
  }));
  const per = (id: string) => split.perDeployment.get(id) ?? [];
  const req = (id: string) => split.requestPoints.get(id) ?? [];
  const value = (id: string, i: number): number => {
    switch (detail) {
      case "requests":
        return STATUS_SERIES.reduce((acc, st) => acc + (req(id)[i]?.[st.key] ?? 0), 0);
      case "errors": {
        const p = req(id)[i];
        const total = STATUS_SERIES.reduce((acc, st) => acc + (p?.[st.key] ?? 0), 0);
        return total > 0 ? ((p?.["5xx"] ?? 0) / total) * 100 : 0;
      }
      case "transfer":
        return per(id)[i]?.egress ?? 0;
      case "cpu":
        return per(id)[i]?.cpu ?? 0;
      case "memory":
        return per(id)[i]?.memory ?? 0;
      case "disk":
        return per(id)[i]?.disk ?? 0;
      case "latency": {
        const rows = bundle.latency.data?.points ?? [];
        const x = bundle.range.startMs + i * b * 1000;
        return rows.find((r) => r.series === id && r.x === x)?.p99 ?? 0;
      }
    }
  };
  const points: SeriesPoint[] = pivot([], bundle.range, []).map((p, i) => {
    const row: SeriesPoint = { x: p.x };
    for (const id of selected) {
      row[id] = value(id, i);
    }
    return row;
  });
  const isCounter = detail === "requests" || detail === "transfer";
  const stats: Shaped["stats"] = (() => {
    switch (detail) {
      case "requests": {
        const total = selected.reduce((acc, id) => acc + sumSeries(points, id), 0);
        const errors = selected.reduce((acc, id) => acc + sumSeries(req(id), "5xx"), 0);
        return [
          { label: "Total", value: formatCount(total) },
          { label: "5xx", value: formatPercent(total > 0 ? (errors / total) * 100 : 0) },
        ];
      }
      case "transfer":
        return [
          {
            label: "Outgoing",
            value: formatBytes(selected.reduce((acc, id) => acc + sumSeries(points, id), 0)),
          },
        ];
      case "cpu": {
        const cpuSeconds = (bundle.cpu.data?.points ?? [])
          .filter((r) => selected.includes(r.series))
          .reduce((acc, r) => acc + r.y / 1e6, 0);
        return [{ label: "CPU time", value: formatCpuTime(cpuSeconds) }];
      }
      case "memory":
        return [
          {
            label: "Peak",
            value: formatBytes(Math.max(0, ...selected.map((id) => maxSeries(points, id)))),
          },
        ];
      case "disk":
        return [
          {
            label: "Peak used",
            value: formatBytes(Math.max(0, ...selected.map((id) => maxSeries(points, id)))),
          },
        ];
      case "latency":
        return [
          {
            label: "p99",
            value: formatMs(
              selected.reduce((acc, id) => acc + meanSeries(points, id), 0) / selected.length,
            ),
          },
        ];
      case "errors": {
        const total = selected.reduce(
          (acc, id) => acc + STATUS_SERIES.reduce((a, st) => a + sumSeries(req(id), st.key), 0),
          0,
        );
        const errors = selected.reduce((acc, id) => acc + sumSeries(req(id), "5xx"), 0);
        return [
          { label: "5xx", value: formatCount(errors) },
          { label: "Rate", value: formatPercent(total > 0 ? (errors / total) * 100 : 0) },
        ];
      }
    }
  })();
  return {
    ...base,
    points,
    series,
    kind: isCounter ? "bar" : "line",
    stacked: isCounter,
    stats,
    yMax:
      detail === "errors"
        ? Math.max(1, ...points.flatMap((p) => selected.map((id) => p[id] ?? 0))) * 1.15
        : undefined,
  };
}

function Overview({
  bundle,
  onOpen,
  selected,
  colorOf,
}: {
  bundle: MetricsBundle;
  onOpen: (d: Detail) => void;
  selected: string[];
  colorOf: (id: string) => string;
}) {
  const domain: [number, number] = [bundle.range.startMs, bundle.range.endMs];
  const index = useDeploymentIndex();
  const split = selected.length > 0 ? splitByDeployment(bundle) : null;
  return (
    <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
      {DETAILS.map((d) => {
        const s = split
          ? shapeSelected(bundle, d, split, selected, colorOf, index)
          : shape(bundle, d);
        return (
          <OverviewCard key={d} title={DETAIL_TITLES[d]} stats={s.stats} onOpen={() => onOpen(d)}>
            <MetricsChart
              points={s.points}
              series={s.series}
              kind={s.kind}
              stacked={s.stacked}
              height={150}
              domain={domain}
              bucketSeconds={bundle.range.bucketSeconds}
              markers={bundle.markers}
              showMarkerLabels={false}
              syncId="app-metrics"
              formatValue={s.format}
              formatTick={s.tick}
              yMax={s.yMax}
              isLoading={s.isLoading}
              isError={s.isError}
            />
          </OverviewCard>
        );
      })}
    </div>
  );
}

function OverviewCard({
  title,
  stats,
  onOpen,
  children,
}: {
  title: string;
  stats: Shaped["stats"];
  onOpen: () => void;
  children: ReactNode;
}) {
  return (
    <section className="border border-gray-4 bg-grayA-1 rounded-lg flex flex-col min-w-0">
      <button
        type="button"
        onClick={onOpen}
        className="group flex items-center justify-between gap-3 px-4 pt-3 pb-1 text-left rounded-t-lg hover:bg-grayA-2 transition-colors"
      >
        <span className="text-[13px] font-medium text-gray-12">{title}</span>
        <ChevronRight
          iconSize="sm-regular"
          className="text-gray-9 group-hover:text-gray-12 transition-colors"
        />
      </button>
      <div className="flex items-baseline gap-5 px-4 pb-2">
        {stats.map((st) => (
          <div key={st.label} className="flex flex-col">
            <span className="text-[11px] text-gray-10">{st.label}</span>
            <span className="flex items-center gap-1.5 text-[15px] font-medium text-gray-12 tabular-nums">
              {st.color && (
                <span className="size-2 rounded-full" style={{ backgroundColor: st.color }} />
              )}
              {st.value}
            </span>
          </div>
        ))}
      </div>
      <div className="px-2 pb-2">{children}</div>
    </section>
  );
}

// ─── Detail ──────────────────────────────────────────────────────────

type Row = {
  id: string;
  info: DeploymentInfo | undefined;
  color: string;
  requests: number;
  errors: number;
  egress: number;
  cpuSeconds: number;
  peakMemory: number;
  peakDisk: number;
  p99: number;
  spark: SeriesPoint[];
  sparkKey: string;
};

// Splits every series by deployment id. Request series arrive as
// "deployment:2xx", so their id is the part before the colon.
function splitByDeployment(bundle: MetricsBundle): {
  ids: string[];
  perDeployment: Map<string, SeriesPoint[]>;
  requestPoints: Map<string, SeriesPoint[]>;
} {
  const ids: string[] = [];
  const seen = new Set<string>();
  const push = (id: string) => {
    if (!seen.has(id)) {
      seen.add(id);
      ids.push(id);
    }
  };
  for (const r of bundle.requests.data?.points ?? []) {
    push(r.series.split(":")[0]);
  }
  for (const q of [bundle.egress, bundle.cpu, bundle.memory, bundle.disk, bundle.ingress]) {
    for (const k of seriesKeysOf(q.data?.points ?? [])) {
      push(k);
    }
  }

  const perDeployment = new Map<string, SeriesPoint[]>();
  const requestPoints = new Map<string, SeriesPoint[]>();
  const b = bundle.range.bucketSeconds;
  for (const id of ids) {
    const only = (q: MetricsBundle["cpu"], key: string, scale: (y: number) => number = (y) => y) =>
      pivot(
        (q.data?.points ?? []).filter((r) => r.series === id).map((r) => ({ ...r, series: key })),
        bundle.range,
        [key],
        scale,
      );
    const egress = only(bundle.egress, "egress");
    const ingress = only(bundle.ingress, "ingress");
    const cpu = only(bundle.cpu, "cpu", (y) => y / b / 1000);
    const memory = only(bundle.memory, "memory");
    const disk = only(bundle.disk, "disk");
    const merged = egress.map((p, i) => ({
      ...p,
      ingress: ingress[i]?.ingress ?? 0,
      cpu: cpu[i]?.cpu ?? 0,
      memory: memory[i]?.memory ?? 0,
      disk: disk[i]?.disk ?? 0,
    }));
    perDeployment.set(id, merged);

    const reqs = pivot(
      (bundle.requests.data?.points ?? [])
        .filter((r) => r.series.startsWith(`${id}:`))
        .map((r) => ({ ...r, series: r.series.split(":")[1] })),
      bundle.range,
      STATUS_SERIES.map((s) => s.key),
    );
    requestPoints.set(id, reqs);
  }
  return { ids, perDeployment, requestPoints };
}

function DetailView({
  detail,
  bundle,
  appId,
  selected,
  colorOf,
}: {
  detail: Detail;
  bundle: MetricsBundle;
  appId: string;
  selected: string[];
  colorOf: (id: string) => string;
}) {
  const index = useDeploymentIndex();
  const all = splitByDeployment(bundle);
  const ids = selected.length > 0 ? all.ids.filter((id) => selected.includes(id)) : all.ids;
  const { perDeployment, requestPoints } = all;
  const b = bundle.range.bucketSeconds;
  const domain: [number, number] = [bundle.range.startMs, bundle.range.endMs];
  const latencyByDep = new Map<string, Map<number, number>>();
  for (const r of bundle.latency.data?.points ?? []) {
    if (!latencyByDep.has(r.series)) {
      latencyByDep.set(r.series, new Map());
    }
    latencyByDep.get(r.series)?.set(r.x, r.p99);
  }

  const rows: Row[] = ids.map((id) => {
    const res = perDeployment.get(id) ?? [];
    const req = requestPoints.get(id) ?? [];
    const requests = STATUS_SERIES.reduce((acc, s) => acc + sumSeries(req, s.key), 0);
    const errors = sumSeries(req, "5xx");
    const cpuSeconds = (bundle.cpu.data?.points ?? [])
      .filter((r) => r.series === id)
      .reduce((acc, r) => acc + r.y / 1e6, 0);
    const latencyMap = latencyByDep.get(id);
    const p99Values = [...(latencyMap?.values() ?? [])].filter((v) => v > 0);
    const p99 = p99Values.length ? p99Values.reduce((a, v) => a + v, 0) / p99Values.length : 0;

    let spark: SeriesPoint[];
    let sparkKey: string;
    switch (detail) {
      case "requests":
        spark = req.map((p) => ({
          x: p.x,
          v: STATUS_SERIES.reduce((acc, s) => acc + (p[s.key] ?? 0), 0),
        }));
        sparkKey = "v";
        break;
      case "errors":
        spark = req.map((p) => ({ x: p.x, v: p["5xx"] ?? 0 }));
        sparkKey = "v";
        break;
      case "latency":
        spark = pivot([], bundle.range, ["v"]).map((p) => ({
          x: p.x,
          v: latencyMap?.get(p.x) ?? 0,
        }));
        sparkKey = "v";
        break;
      case "transfer":
        spark = res;
        sparkKey = "egress";
        break;
      default:
        spark = res;
        sparkKey = detail;
    }

    return {
      id,
      info: index.get(id),
      color: colorOf(id),
      requests,
      errors,
      egress: sumSeries(res, "egress"),
      cpuSeconds,
      peakMemory: maxSeries(res, "memory"),
      peakDisk: maxSeries(res, "disk"),
      p99,
      spark,
      sparkKey,
    };
  });

  const sortValue = (r: Row): number => {
    switch (detail) {
      case "requests":
        return r.requests;
      case "errors":
        return r.errors;
      case "transfer":
        return r.egress;
      case "cpu":
        return r.cpuSeconds;
      case "memory":
        return r.peakMemory;
      case "disk":
        return r.peakDisk;
      case "latency":
        return r.p99;
    }
  };
  rows.sort((a, c) => sortValue(c) - sortValue(a));

  // Big chart: one series per deployment. Counters stack (they add up to
  // the app total), gauges and latency overlay.
  const series: SeriesKey[] = rows.map((r) => ({
    key: r.id,
    label: r.info ? `${shortSha(r.info.sha) || r.id} ${r.info.message ?? ""}`.trim() : r.id,
    color: r.color,
  }));
  const bigPoints: SeriesPoint[] = pivot([], bundle.range, []).map((p, i) => {
    const row: SeriesPoint = { x: p.x };
    for (const r of rows) {
      row[r.id] = r.spark[i]?.[r.sparkKey] ?? 0;
    }
    return row;
  });
  const isCounter =
    detail === "requests" || detail === "errors" || detail === "transfer" || detail === "cpu";
  const [format, tick] = ((): [(v: number) => string, (v: number) => string] => {
    switch (detail) {
      case "requests":
      case "errors":
        return [formatCount, tickCount];
      case "transfer":
        return [(v: number) => formatBytes(v), tickBytes];
      case "cpu":
        return [formatVcpu, tickVcpu];
      case "latency":
        return [formatMs, tickMs];
      default:
        return [(v: number) => formatBytes(v), tickBytes];
    }
  })();
  const anyLoading = [bundle.requests, bundle.egress, bundle.cpu, bundle.memory, bundle.disk].some(
    (q) => q.isLoading,
  );

  return (
    <div className="flex flex-col gap-4">
      <section className="border border-gray-4 bg-grayA-1 rounded-lg">
        <header className="flex items-center justify-between px-4 pt-3 pb-1">
          <h3 className="text-[13px] font-medium text-gray-12">
            {DETAIL_TITLES[detail]} <span className="text-gray-10 font-normal">by deployment</span>
          </h3>
          <span className="text-[11px] text-gray-10">
            {detail === "cpu" ? "vCPU per bucket" : detail === "latency" ? "p99" : null}
          </span>
        </header>
        <div className="px-2 pb-2">
          <MetricsChart
            points={bigPoints}
            series={series}
            kind={isCounter ? "bar" : "line"}
            stacked={isCounter}
            height={260}
            domain={domain}
            bucketSeconds={b}
            markers={bundle.markers}
            showMarkerLabels={false}
            formatValue={format}
            formatTick={tick}
            isLoading={anyLoading}
            isError={bundle.requests.isError}
          />
        </div>
      </section>

      <BreakdownTable rows={rows} detail={detail} appId={appId} bundle={bundle} />
    </div>
  );
}

function BreakdownTable({
  rows,
  detail,
  appId,
  bundle,
}: {
  rows: Row[];
  detail: Detail;
  appId: string;
  bundle: MetricsBundle;
}) {
  const { projectId } = useProjectData();
  const workspace = useWorkspaceNavigation();
  const domain: [number, number] = [bundle.range.startMs, bundle.range.endMs];
  const primary = (r: Row): string => {
    switch (detail) {
      case "requests":
        return formatCount(r.requests);
      case "errors":
        return formatCount(r.errors);
      case "transfer":
        return formatBytes(r.egress);
      case "cpu":
        return formatCpuTime(r.cpuSeconds);
      case "memory":
        return formatBytes(r.peakMemory);
      case "disk":
        return formatBytes(r.peakDisk);
      case "latency":
        return formatMs(r.p99);
    }
  };

  const cols: { key: string; label: string; cell: (r: Row) => string }[] = [
    { key: "requests", label: "Requests", cell: (r) => formatCount(r.requests) },
    {
      key: "errors",
      label: "5xx",
      cell: (r) => (r.requests > 0 ? formatPercent((r.errors / r.requests) * 100) : "0%"),
    },
    { key: "egress", label: "Egress", cell: (r) => formatBytes(r.egress) },
    { key: "cpu", label: "CPU time", cell: (r) => formatCpuTime(r.cpuSeconds) },
    { key: "memory", label: "Peak mem", cell: (r) => formatBytes(r.peakMemory) },
  ];

  return (
    <section className="border border-gray-4 bg-grayA-1 rounded-lg overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-[12px]">
          <thead>
            <tr className="text-left text-gray-10 border-b border-gray-4">
              <th className="font-normal px-4 py-2">Deployment</th>
              {cols.map((c) => (
                <th
                  key={c.key}
                  className={cn(
                    "font-normal px-3 py-2 text-right whitespace-nowrap",
                    c.key === detail && "text-gray-12",
                  )}
                >
                  {c.label}
                </th>
              ))}
              <th className="font-normal px-3 py-2 text-right whitespace-nowrap text-gray-12">
                {DETAIL_TITLES[detail]}
              </th>
              <th className="px-3 py-2 w-[140px]" />
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && (
              <tr>
                <td colSpan={cols.length + 3} className="px-4 py-6 text-center text-gray-10">
                  No deployments served traffic in this range.
                </td>
              </tr>
            )}
            {rows.map((r) => (
              <tr key={r.id} className="border-b border-gray-3 last:border-0 hover:bg-grayA-2">
                <td className="px-4 py-2.5 min-w-[260px]">
                  <Link
                    href={routes.projects.apps.deployment({
                      workspaceSlug: workspace.slug,
                      projectId,
                      appId,
                      deploymentId: r.id,
                    })}
                    className="flex items-center gap-2.5 group"
                  >
                    <span
                      className="size-2 rounded-full shrink-0"
                      style={{ backgroundColor: r.color }}
                    />
                    <span className="font-mono text-gray-12">
                      {shortSha(r.info?.sha) || r.id.slice(-7)}
                    </span>
                    <span className="text-gray-11 truncate max-w-[280px] group-hover:text-gray-12">
                      {r.info?.message ?? r.id}
                    </span>
                    {r.info && (
                      <Badge variant="secondary" size="sm" className="capitalize shrink-0">
                        {r.info.status}
                      </Badge>
                    )}
                    {r.info && (
                      <span className="text-gray-9 tabular-nums shrink-0">
                        {new Date(r.info.createdAt).toLocaleDateString(undefined, {
                          month: "short",
                          day: "numeric",
                        })}
                      </span>
                    )}
                  </Link>
                </td>
                {cols.map((c) => (
                  <td
                    key={c.key}
                    className={cn(
                      "px-3 py-2.5 text-right font-mono tabular-nums text-gray-11",
                      c.key === detail && "text-gray-12",
                    )}
                  >
                    {c.cell(r)}
                  </td>
                ))}
                <td className="px-3 py-2.5 text-right font-mono tabular-nums text-gray-12 font-medium">
                  {primary(r)}
                </td>
                <td className="px-3 py-1.5">
                  <MetricsChart
                    points={r.spark}
                    series={[{ key: r.sparkKey, label: DETAIL_TITLES[detail], color: r.color }]}
                    kind="area"
                    height={28}
                    domain={domain}
                    bucketSeconds={bundle.range.bucketSeconds}
                    showAxes={false}
                    formatValue={() => ""}
                    emptyMessage=""
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
