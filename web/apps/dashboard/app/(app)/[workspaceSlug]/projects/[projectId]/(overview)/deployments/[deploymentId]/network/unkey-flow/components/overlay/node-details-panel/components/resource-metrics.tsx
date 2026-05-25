"use client";

import { trpc } from "@/lib/trpc/client";
import { useDeployment } from "../../../../../../layout-provider";
import {
  bytesToMib,
  formatBytesPerSecondParts,
  formatCpuParts,
  formatMemoryParts,
  formatStorageParts,
  formatTooltipPercent,
} from "@/lib/utils/deployment-formatters";
import type { TimeWindow } from "@unkey/clickhouse";
import {
  ArrowOppositeDirectionY,
  ChevronExpandY,
  Grid,
  Harddrive,
  Microchip,
  Ram,
} from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import {
  type AreaChartPoint,
  AreaTimeseriesChart,
  formatYAxisCompactBytesPerSecond,
} from "./chart/area-timeseries-chart";

// Dashboard panel refreshes every 3s. The chart's live tip reads from raw
// FINAL so new checkpoints (5s heimdall cadence) show up at most one
// refresh after they land. Each refresh fires 7 queries (cpu, memory,
// disk, instances, network ingress, network egress, summary), each
// scanning at most ~1 minute of raw data per workspace+resource. ~2.3 q/s
// per open panel, well within budget.
const REFETCH_INTERVAL_MS = 3_000;

const WINDOW_LABELS: Record<TimeWindow, string> = {
  "15m": "Past 15 minutes",
  "1h": "Past hour",
  "3h": "Past 3 hours",
  "6h": "Past 6 hours",
  "12h": "Past 12 hours",
  "1d": "Past day",
  "1w": "Past week",
  "30d": "Past 30 days",
  "90d": "Past 90 days",
  "1y": "Past year",
};
const WINDOW_OPTIONS: TimeWindow[] = [
  "15m",
  "1h",
  "3h",
  "6h",
  "12h",
  "1d",
  "1w",
  "30d",
  "90d",
  "1y",
];

// Windows that span across days: tooltip needs the date or "4:14 AM" is
// ambiguous. 12h doesn't technically span a day, but a user hovering at
// 6 AM can't tell if they're looking at today or yesterday without the date.
const WINDOWS_NEEDING_DATE: TimeWindow[] = ["12h", "1d", "1w", "30d", "90d", "1y"];

// Window → duration in millis. We anchor the chart's x-axis domain to
// this range so a "Past day" axis always reads as 24h wide even when the
// deployment is only a few hours old — otherwise switching between "Past
// hour" and "Past day" on sparse data produces identical-looking axes.
// Mirrors the server-side WINDOW_CONFIG in @unkey/clickhouse/src/resources.ts
// (kept in sync manually; they're small and rarely change).
const WINDOW_MS: Record<TimeWindow, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "3h": 3 * 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "12h": 12 * 60 * 60 * 1000,
  "1d": 24 * 60 * 60 * 1000,
  "1w": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
  "90d": 90 * 24 * 60 * 60 * 1000,
  "1y": 365 * 24 * 60 * 60 * 1000,
};

// ─── main ─────────────────────────────────────────────────────────────

type ResourceMetricsProps = {
  resourceId: string;
  // When > 0 the deployment has ephemeral storage provisioned, so the disk
  // chart is worth showing. For disk-less deployments we skip it entirely,
  // avoiding a flat-line chart with no data to read.
  storageMib?: number;
  // K8s pod name. Set when the panel is scoped to a single instance; all
  // queries filter `instance_id = instanceName` so the charts show that
  // replica's metrics, not the deployment-wide sum across replicas.
  instanceName?: string;
};

export function ResourceMetrics({ resourceId, storageMib, instanceName }: ResourceMetricsProps) {
  const { deployment } = useDeployment();
  const params = { resourceId, instanceName };
  const [window, setWindow] = useState<TimeWindow>("1h");
  const showDateInTooltip = WINDOWS_NEEDING_DATE.includes(window);
  const diskEnabled = (storageMib ?? 0) > 0;
  const isInstanceScoped = Boolean(instanceName);

  // keepPreviousData holds the prior window's data visible during the
  // refetch after a window change, so the charts don't collapse to their
  // loading skeleton (different height) and snap back — which would jar
  // the whole panel up/down on every selection.
  const cpu = trpc.deploy.metrics.getDeploymentCpuTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, keepPreviousData: true },
  );
  const memory = trpc.deploy.metrics.getDeploymentMemoryTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, keepPreviousData: true },
  );
  const disk = trpc.deploy.metrics.getDeploymentDiskTimeseries.useQuery(
    { ...params, window },
    {
      refetchInterval: REFETCH_INTERVAL_MS,
      enabled: diskEnabled,
      keepPreviousData: true,
    },
  );
  const networkEgress = trpc.deploy.metrics.getDeploymentNetworkEgressTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, keepPreviousData: true },
  );
  const networkIngress = trpc.deploy.metrics.getDeploymentNetworkIngressTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, keepPreviousData: true },
  );
  const instances = trpc.deploy.metrics.getDeploymentInstanceCountTimeseries.useQuery(
    { ...params, window },
    {
      refetchInterval: REFETCH_INTERVAL_MS,
      enabled: !isInstanceScoped,
      keepPreviousData: true,
    },
  );
  // Bar reads the chart's right-edge bucket so the two stay in lockstep —
  // a rolling-average "current" smooths bursts, which made a 72% chart peak
  // sit next to a 30% bar and read as a contradiction.
  const cpuUsedMilli = Math.round(cpu.data?.at(-1)?.y ?? 0);
  const cpuAllocatedMilli = deployment.cpuMillicores;
  const memUsedBytes = memory.data?.at(-1)?.y ?? 0;
  const memAllocatedBytes = deployment.memoryMib * 1024 * 1024;
  const diskUsedBytes = disk.data?.at(-1)?.y ?? 0;
  const diskAllocatedBytes = (storageMib ?? 0) * 1024 * 1024;
  const instanceCount = Math.round(instances.data?.at(-1)?.y ?? 0);

  const nowMs = Date.now();
  const xAxisDomain: [number, number] = [nowMs - WINDOW_MS[window], nowMs];

  const [isWindowTransition, setIsWindowTransition] = useState(false);
  const prevWindowRef = useRef(window);
  useEffect(() => {
    if (prevWindowRef.current !== window) {
      prevWindowRef.current = window;
      setIsWindowTransition(true);
    }
  }, [window]);
  const anyFetching =
    cpu.isFetching ||
    memory.isFetching ||
    disk.isFetching ||
    networkEgress.isFetching ||
    networkIngress.isFetching ||
    instances.isFetching;
  useEffect(() => {
    if (isWindowTransition && !anyFetching) {
      setIsWindowTransition(false);
    }
  }, [isWindowTransition, anyFetching]);

  return (
    <div>
      <div className="flex items-center gap-3 px-4 pt-4 w-full">
        <div className="text-gray-10 text-xs whitespace-nowrap">Runtime metrics</div>
        <div className="flex-1 min-w-0 border-t border-grayA-3" />
        <Select value={window} onValueChange={(v) => setWindow(v as TimeWindow)}>
          <SelectTrigger
            wrapperClassName="w-fit shrink-0"
            className="h-7 min-h-0! rounded-lg border-grayA-4 bg-transparent shadow-sm text-gray-12 text-xs focus:ring-0"
            rightIcon={<ChevronExpandY className="absolute right-2.5 text-gray-9 size-3" />}
          >
            <SelectValue className="text-xs" />
          </SelectTrigger>
          <SelectContent className="min-w-[160px]">
            {WINDOW_OPTIONS.map((option) => (
              <SelectItem
                key={option}
                value={option}
                className="cursor-pointer hover:bg-grayA-3 data-highlighted:bg-grayA-2 text-xs"
              >
                {WINDOW_LABELS[option]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {!isInstanceScoped && (
        <InstancesSection
          points={instances.data}
          currentCount={instanceCount}
          isLoading={instances.isLoading || isWindowTransition}
          isError={instances.isError}
          showDateInTooltip={showDateInTooltip}
          xAxisDomain={xAxisDomain}
        />
      )}

      <CpuSection
        points={cpu.data}
        usedMilli={cpuUsedMilli}
        allocatedMilli={cpuAllocatedMilli}
        isLoading={cpu.isLoading || isWindowTransition}
        isError={cpu.isError}
        showDateInTooltip={showDateInTooltip}
        xAxisDomain={xAxisDomain}
        isFirst={isInstanceScoped}
      />

      <MemorySection
        points={memory.data}
        usedBytes={memUsedBytes}
        allocatedBytes={memAllocatedBytes}
        isLoading={memory.isLoading || isWindowTransition}
        isError={memory.isError}
        showDateInTooltip={showDateInTooltip}
        xAxisDomain={xAxisDomain}
      />

      {diskEnabled && (
        <DiskSection
          points={disk.data}
          usedBytes={diskUsedBytes}
          allocatedBytes={diskAllocatedBytes}
          isLoading={disk.isLoading || isWindowTransition}
          isError={disk.isError}
          showDateInTooltip={showDateInTooltip}
          xAxisDomain={xAxisDomain}
        />
      )}

      <NetworkSection
        egressPoints={networkEgress.data}
        ingressPoints={networkIngress.data}
        isLoading={networkEgress.isLoading || networkIngress.isLoading || isWindowTransition}
        isError={networkEgress.isError || networkIngress.isError}
        showDateInTooltip={showDateInTooltip}
        xAxisDomain={xAxisDomain}
      />
    </div>
  );
}

// ─── formatting ───────────────────────────────────────────────────────

// ─── display ──────────────────────────────────────────────────────────

function UtilizationBar({
  used,
  allocated,
  color,
  usedLabel,
  allocatedLabel,
}: {
  used: number;
  allocated: number;
  color: string;
  usedLabel: string;
  allocatedLabel: string;
}) {
  const ratio = allocated > 0 ? Math.min(used / allocated, 1) : 0;
  const pct = Math.round(ratio * 100);

  return (
    <div className="flex items-center gap-2 text-[12px] tabular-nums">
      <span className="text-gray-12 font-medium">{usedLabel}</span>
      <span className="text-grayA-9">/</span>
      <span className="text-grayA-9">{allocatedLabel}</span>
      <div className="w-[80px] h-2 rounded-full overflow-hidden shadow-[inset_0_1px_2px_rgba(0,0,0,0.15)] dark:shadow-[inset_0_1px_2px_rgba(255,255,255,0.08)] bg-grayA-2 dark:bg-grayA-3">
        <div
          className="h-full rounded-full transition-all duration-500 ease-out"
          style={{
            width: `${pct}%`,
            backgroundColor: color,
            boxShadow: "inset 0 1px 0 rgba(255,255,255,0.15)",
          }}
        />
      </div>
      <span className="text-gray-11 font-medium">{pct}%</span>
    </div>
  );
}

// NetworkSection renders egress + ingress as a single stacked area chart
// with a Y-axis, plus a summary row above showing peak + total for each
// direction. Peak is the largest rate in any bucket; total is the integral
// of the rate series (left-Riemann over actual inter-point dt, so it works
// for any bucket size the current window picked).
type NetworkSectionProps = {
  egressPoints: Array<{ x: number; y: number }> | undefined;
  ingressPoints: Array<{ x: number; y: number }> | undefined;
  isLoading: boolean;
  isError?: boolean;
  showDateInTooltip?: boolean;
  xAxisDomain: [number, number];
};

const EGRESS_COLOR = "hsl(var(--error-8))";
const INGRESS_COLOR = "hsl(var(--success-8))";

function NetworkSection({
  egressPoints,
  ingressPoints,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
}: NetworkSectionProps) {
  const egress = summarizeRateSeries(egressPoints);
  const ingress = summarizeRateSeries(ingressPoints);
  const data = mergeNetworkSeries(egressPoints, ingressPoints);
  return (
    <div className="flex flex-col gap-3 px-4 w-full border-t border-grayA-3 pt-6 mt-2">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-error-3 text-error-11 rounded-md size-[22px] items-center flex justify-center">
          <ArrowOppositeDirectionY iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-12 text-[13px]">Network</span>
        <div className="ml-auto flex items-center gap-3 text-[12px] tabular-nums">
          <span className="flex items-center gap-1">
            <span className="text-error-9">↑</span>
            <span className="text-gray-12 font-medium">
              {formatBytesPerSecondParts(egress.peak).value}
            </span>
            <span className="text-grayA-10">{formatBytesPerSecondParts(egress.peak).unit}</span>
          </span>
          <span className="flex items-center gap-1">
            <span className="text-success-9">↓</span>
            <span className="text-gray-12 font-medium">
              {formatBytesPerSecondParts(ingress.peak).value}
            </span>
            <span className="text-grayA-10">{formatBytesPerSecondParts(ingress.peak).unit}</span>
          </span>
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{
          network_ingress: { label: "Ingress", color: INGRESS_COLOR },
          network_egress: { label: "Egress", color: EGRESS_COLOR },
        }}
        height={200}
        isLoading={isLoading}
        isError={isError}
        showDateInTooltip={showDateInTooltip}
        xAxisDomain={xAxisDomain}
        formatYTick={formatYAxisCompactBytesPerSecond}
      />
    </div>
  );
}

const INSTANCES_COLOR = "hsl(var(--error-8))";
const CPU_COLOR = "hsl(var(--feature-8))";
const MEMORY_COLOR = "hsl(var(--info-8))";
const DISK_COLOR = "hsl(var(--warning-8))";

type InstancesSectionProps = {
  points: Array<{ x: number; y: number }> | undefined;
  currentCount: number;
  isLoading: boolean;
  isError?: boolean;
  showDateInTooltip?: boolean;
  xAxisDomain: [number, number];
};

// InstancesSection renders active-instance count as an area chart so it
// matches the other rows. Count is discrete (integers), so the y-axis
// uses a whole-number formatter and a small axis floor keeps the line
// visible even for single-replica deployments.
function InstancesSection({
  points,
  currentCount,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
}: InstancesSectionProps) {
  const data: AreaChartPoint[] = (points ?? []).map((p) => ({
    originalTimestamp: p.x,
    active_instances: p.y,
  }));
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-6">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-error-3 text-error-11 rounded-md size-[22px] items-center flex justify-center">
          <Grid iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-12 text-[13px]">Active instances</span>
        <div className="ml-auto">
          <span className="text-gray-12 font-medium text-[13px] tabular-nums">
            {currentCount}
            <span className="font-normal text-grayA-10"> vm</span>
          </span>
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ active_instances: { label: "Instances", color: INSTANCES_COLOR } }}
        height={200}
        isLoading={isLoading}
        isError={isError}
        showDateInTooltip={showDateInTooltip}
        formatTooltipValue={(count) => ({ value: `${Math.round(count)}`, unit: "vm" })}
        formatYTick={formatInstanceTick}
        // 3 keeps the axis showing 0 / 1 / 2 / 3 on single-replica deployments
        // instead of collapsing everything onto the 0 line.
        axisFloor={3}
        xAxisDomain={xAxisDomain}
      />
    </div>
  );
}

// Integer Y-axis ticks for instance counts. No K/M/G scaling — you don't
// run 1024 pods on a deployment and want to read "1K".
function formatInstanceTick(v: number): string {
  return `${Math.round(v)}`;
}

type CpuSectionProps = {
  points: Array<{ x: number; y: number }> | undefined;
  usedMilli: number;
  allocatedMilli: number;
  isLoading: boolean;
  isError?: boolean;
  showDateInTooltip?: boolean;
  xAxisDomain: [number, number];
  isFirst?: boolean;
};

function CpuSection({
  points,
  usedMilli,
  allocatedMilli,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
  isFirst,
}: CpuSectionProps) {
  const data: AreaChartPoint[] = (points ?? []).map((p) => ({
    originalTimestamp: p.x,
    cpu_usage: p.y,
  }));
  return (
    <div
      className={
        isFirst
          ? "flex flex-col gap-3 px-4 w-full mt-6"
          : "flex flex-col gap-3 px-4 w-full border-t border-grayA-3 pt-6 mt-2"
      }
    >
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-feature-3 text-feature-11 rounded-md size-[22px] items-center flex justify-center">
          <Microchip iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-12 text-[13px]">CPU usage</span>
        <div className="ml-auto">
          <UtilizationBar
            used={usedMilli}
            allocated={allocatedMilli}
            color={CPU_COLOR}
            usedLabel={`${usedMilli}m`}
            allocatedLabel={`${formatCpuParts(allocatedMilli).value} ${formatCpuParts(allocatedMilli).unit}`}
          />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ cpu_usage: { label: "CPU", color: CPU_COLOR } }}
        height={200}
        isLoading={isLoading}
        isError={isError}
        showDateInTooltip={showDateInTooltip}
        formatTooltipValue={(millicores) => {
          if (allocatedMilli <= 0) {
            return { value: `${Math.round(millicores)}`, unit: "m" };
          }
          const pct = formatTooltipPercent((millicores / allocatedMilli) * 100);
          return { value: pct, hint: `${Math.round(millicores)}m` };
        }}
        axisFloor={100}
        formatYTick={(v) => {
          if (v === 0 || allocatedMilli <= 0) {
            return "";
          }
          return `${Math.round((v / allocatedMilli) * 100)}%`;
        }}
        xAxisDomain={xAxisDomain}
      />
    </div>
  );
}

type MemorySectionProps = {
  points: Array<{ x: number; y: number }> | undefined;
  usedBytes: number;
  allocatedBytes: number;
  isLoading: boolean;
  isError?: boolean;
  showDateInTooltip?: boolean;
  xAxisDomain: [number, number];
};

// MemorySection renders memory-bytes timeseries as an area chart. Tooltip
// shows "43 MiB (17%)" — value+unit for the absolute figure, hint for the
// utilization %.
function MemorySection({
  points,
  usedBytes,
  allocatedBytes,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
}: MemorySectionProps) {
  const data: AreaChartPoint[] = (points ?? []).map((p) => ({
    originalTimestamp: p.x,
    memory_usage: p.y,
  }));
  return (
    <div className="flex flex-col gap-3 px-4 w-full border-t border-grayA-3 pt-6 mt-2">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-info-3 text-info-11 rounded-md size-[22px] items-center flex justify-center">
          <Ram iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-12 text-[13px]">Memory usage</span>
        <div className="ml-auto">
          <UtilizationBar
            used={usedBytes}
            allocated={allocatedBytes}
            color={MEMORY_COLOR}
            usedLabel={`${formatMemoryParts(bytesToMib(usedBytes)).value} ${formatMemoryParts(bytesToMib(usedBytes)).unit}`}
            allocatedLabel={`${formatMemoryParts(bytesToMib(allocatedBytes)).value} ${formatMemoryParts(bytesToMib(allocatedBytes)).unit}`}
          />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ memory_usage: { label: "Memory", color: MEMORY_COLOR } }}
        height={200}
        isLoading={isLoading}
        isError={isError}
        showDateInTooltip={showDateInTooltip}
        formatTooltipValue={(bytes) => {
          const parts = formatMemoryParts(bytesToMib(bytes));
          if (allocatedBytes <= 0) {
            return { value: parts.value, unit: parts.unit };
          }
          return {
            value: parts.value,
            unit: parts.unit,
            hint: `(${formatTooltipPercent((bytes / allocatedBytes) * 100)})`,
          };
        }}
        axisFloor={1024 * 1024}
        xAxisDomain={xAxisDomain}
      />
    </div>
  );
}

type DiskSectionProps = {
  points: Array<{ x: number; y: number }> | undefined;
  usedBytes: number;
  allocatedBytes: number;
  isLoading: boolean;
  isError?: boolean;
  showDateInTooltip?: boolean;
  xAxisDomain: [number, number];
};

// DiskSection mirrors NetworkSection's layout so the two rows read as
// a matching pair: same header shape (icon + label + right-aligned stats)
// and the same area chart styling below. Unlike network, disk is a level
// metric (not a rate) so we pass a bytes-only tooltip formatter and a
// smaller axis floor (1 MiB) so idle-but-provisioned disks don't render
// with a fake "1 KiB" baseline.
function DiskSection({
  points,
  usedBytes,
  allocatedBytes,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
}: DiskSectionProps) {
  const data: AreaChartPoint[] = (points ?? []).map((p) => ({
    originalTimestamp: p.x,
    disk_usage: p.y,
  }));
  return (
    <div className="flex flex-col gap-3 px-4 w-full border-t border-grayA-3 pt-6 mt-2">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-warning-3 text-warning-11 rounded-md size-[22px] items-center flex justify-center">
          <Harddrive iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-12 text-[13px]">Disk usage</span>
        <div className="ml-auto">
          <UtilizationBar
            used={usedBytes}
            allocated={allocatedBytes}
            color={DISK_COLOR}
            usedLabel={`${formatStorageParts(bytesToMib(usedBytes)).value} ${formatStorageParts(bytesToMib(usedBytes)).unit}`}
            allocatedLabel={`${formatStorageParts(bytesToMib(allocatedBytes)).value} ${formatStorageParts(bytesToMib(allocatedBytes)).unit}`}
          />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ disk_usage: { label: "Disk", color: DISK_COLOR } }}
        height={200}
        isLoading={isLoading}
        isError={isError}
        showDateInTooltip={showDateInTooltip}
        formatTooltipValue={(bytes) => {
          const parts = formatMemoryParts(bytesToMib(bytes));
          if (allocatedBytes <= 0) {
            return { value: parts.value, unit: parts.unit };
          }
          return {
            value: parts.value,
            unit: parts.unit,
            hint: `(${formatTooltipPercent((bytes / allocatedBytes) * 100)})`,
          };
        }}
        axisFloor={1024 * 1024}
        xAxisDomain={xAxisDomain}
      />
    </div>
  );
}

// mergeNetworkSeries aligns the two rate timeseries by bucket timestamp.
// The two tRPC queries share the same window/bucket config so timestamps
// line up exactly; any stray bucket present in only one series still gets
// rendered (with the other direction as 0) rather than being dropped.
function mergeNetworkSeries(
  egress: Array<{ x: number; y: number }> | undefined,
  ingress: Array<{ x: number; y: number }> | undefined,
): AreaChartPoint[] {
  const byTs = new Map<number, { e: number; i: number }>();
  for (const p of egress ?? []) {
    const cur = byTs.get(p.x) ?? { e: 0, i: 0 };
    cur.e = p.y;
    byTs.set(p.x, cur);
  }
  for (const p of ingress ?? []) {
    const cur = byTs.get(p.x) ?? { e: 0, i: 0 };
    cur.i = p.y;
    byTs.set(p.x, cur);
  }
  return [...byTs.entries()]
    .sort(([a], [b]) => a - b)
    .map(([ts, { e, i }]) => ({
      originalTimestamp: ts,
      network_egress: e,
      network_ingress: i,
    }));
}

// summarizeRateSeries returns the peak byte rate and the integrated total
// bytes across a rate timeseries. Total uses left-Riemann over the actual
// inter-point dt so the same function works for 15s / 1min / 1h buckets.
//
// The last sample needs an explicit bucket width — we don't have a "next"
// timestamp to subtract from for it, and the natural choice is the dt
// implied by the surrounding samples. A naive loop that only accumulates
// gaps between adjacent points would silently drop the last bucket and
// return 0 for any single-point window even when the rate is non-zero.
function summarizeRateSeries(points: Array<{ x: number; y: number }> | undefined): {
  peak: number;
  total: number;
} {
  if (!points?.length) {
    return { peak: 0, total: 0 };
  }
  let peak = 0;
  let total = 0;
  // Median dt is more stable than "last gap" against irregular spacing
  // (lifecycle checkpoints, MV grace zone). For a single-point series we
  // fall back to 0, which still surfaces the peak rate so the caller can
  // render "12 MiB/s · 0 B total" instead of the incorrect 0/0.
  const dts: number[] = [];
  for (let i = 1; i < points.length; i++) {
    dts.push((points[i].x - points[i - 1].x) / 1000);
  }
  const tailDtSec = dts.length > 0 ? dts[Math.floor(dts.length / 2)] : 0;
  for (let i = 0; i < points.length; i++) {
    const y = points[i].y;
    if (y > peak) {
      peak = y;
    }
    const dtSec = i < points.length - 1 ? (points[i + 1].x - points[i].x) / 1000 : tailDtSec;
    total += y * dtSec;
  }
  return { peak, total };
}
