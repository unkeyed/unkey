"use client";

import { trpc } from "@/lib/trpc/client";
import {
  formatBytesPerSecondParts,
  formatCpuParts,
  formatMemoryParts,
  formatNetworkBytesParts,
  formatStorageParts,
} from "@/lib/utils/deployment-formatters";
import type { TimeWindow } from "@unkey/clickhouse";
import {
  ArrowOppositeDirectionY,
  Bolt,
  ChevronExpandY,
  Focus,
  Grid,
  Harddrive,
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

// Heimdall isn't shipped yet, so cpu/memory/disk/network charts would
// render as empty timeseries. Hide them until the metering pipeline lands.
// Queries still fire — the tables exist and return empty, so there's
// nothing to gate at the network layer.
const RUNTIME_METRICS_ENABLED = false;

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
  resourceType: "deployment" | "sentinel";
  resourceId: string;
  // When > 0 the deployment has ephemeral storage provisioned, so the disk
  // chart is worth showing. For sentinel nodes and disk-less deployments
  // we skip it entirely, avoiding a flat-line chart with no data to read.
  storageMib?: number;
  // K8s pod name. Set when the panel is scoped to a single instance; all
  // queries filter `instance_id = instanceName` so the charts show that
  // replica's metrics, not the deployment-wide sum across replicas.
  instanceName?: string;
};

export function ResourceMetrics({
  resourceType,
  resourceId,
  storageMib,
  instanceName,
}: ResourceMetricsProps) {
  const params = { resourceType, resourceId, instanceName };
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
    { refetchInterval: REFETCH_INTERVAL_MS, enabled: diskEnabled, keepPreviousData: true },
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
  const summary = trpc.deploy.metrics.getDeploymentResourceSummary.useQuery(params, {
    refetchInterval: REFETCH_INTERVAL_MS,
  });

  const cpuUsedMilli = Math.round(summary.data?.current_cpu_millicores ?? 0);
  const cpuAllocatedMilli = Math.round(summary.data?.cpu_allocated_millicores ?? 0);
  const memUsedBytes = summary.data?.current_memory_bytes ?? 0;
  const memAllocatedBytes = summary.data?.memory_allocated_bytes ?? 0;
  const diskUsedBytes = summary.data?.current_disk_used_bytes ?? 0;
  const diskAllocatedBytes = (storageMib ?? 0) * 1024 * 1024;
  const instanceCount = summary.data?.active_instances ?? 0;

  // Anchor every chart's x-axis to [now - window, now] so switching
  // "Past hour" → "Past day" visibly re-scales the axis even when data
  // only covers a fraction of the selected window. Recomputed each
  // render; the 3s refetch cadence means the axis slides imperceptibly.
  const nowMs = Date.now();
  const xAxisDomain: [number, number] = [nowMs - WINDOW_MS[window], nowMs];

  // Window-transition loading: distinguishes a user-initiated window
  // switch (show loading skeletons so we don't render stale data on a
  // new axis) from the 3s periodic refetch (keepPreviousData keeps old
  // data visible, no flicker). We flip `isWindowTransition=true` on
  // every window change and clear it once all relevant queries have
  // finished fetching the new window's data.
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

  // Instance-scoped panel hides the instances chart, so with the runtime
  // metrics flag off there's nothing left to render — skip the empty
  // "Runtime metrics" header + window selector entirely.
  if (!RUNTIME_METRICS_ENABLED && isInstanceScoped) {
    return null;
  }

  return (
    <div>
      <div className="flex flex-col px-4 w-full gap-2">
        <div className="flex items-center gap-3 w-full">
          <div className="text-gray-9 text-xs whitespace-nowrap">Runtime metrics</div>
          <div className="h-0.5 bg-grayA-3 rounded-xs flex-1" />
        </div>
        <div className="flex justify-end w-full">
          <Select value={window} onValueChange={(v) => setWindow(v as TimeWindow)}>
            <SelectTrigger
              className="bg-transparent rounded-full flex items-center gap-1 border-0 h-auto min-h-0! p-0! focus:border-none focus:ring-0 hover:bg-grayA-2 transition-colors justify-normal text-gray-11 text-xs"
              rightIcon={<ChevronExpandY className="text-accent-8 size-3" />}
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

      {RUNTIME_METRICS_ENABLED && (
        <>
          <CpuSection
            points={cpu.data}
            usedMilli={cpuUsedMilli}
            allocatedMilli={cpuAllocatedMilli}
            isLoading={cpu.isLoading || isWindowTransition}
            isError={cpu.isError}
            showDateInTooltip={showDateInTooltip}
            xAxisDomain={xAxisDomain}
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
        </>
      )}
    </div>
  );
}

// ─── formatting ───────────────────────────────────────────────────────

function bytesToMib(bytes: number): number {
  if (bytes <= 0) {
    return 0;
  }
  return Math.round(bytes / (1024 * 1024));
}

function percent(used: number, allocated: number): string | null {
  if (!allocated || allocated <= 0) {
    return null;
  }
  return formatTooltipPercent((used / allocated) * 100);
}

// Shared formatter for percent strings so tooltip + header agree.
function formatTooltipPercent(p: number): string {
  if (!Number.isFinite(p) || p <= 0) {
    return "0%";
  }
  return p < 1 ? `${p.toFixed(2)}%` : p < 10 ? `${p.toFixed(1)}%` : `${Math.round(p)}%`;
}

// ─── display ──────────────────────────────────────────────────────────

type CpuUsageValueProps = {
  usedMilli: number;
  allocatedMilli: number;
  percent: string | null;
};

// CPU: show utilization % primary. Raw millicores are noise for most users.
// they want to know "am I approaching my limit?". Allocation shown in the
// friendly fraction format (1/4, 1/2, 1 vCPU) so it's instantly recognizable.
function CpuUsageValue({ usedMilli, allocatedMilli, percent }: CpuUsageValueProps) {
  const allocated = formatCpuParts(allocatedMilli);
  return (
    <div className="flex items-baseline gap-1.5 text-[13px] tabular-nums">
      <span className="text-gray-12 font-medium">{percent ?? `${usedMilli}m`}</span>
      {allocatedMilli > 0 && (
        <span className="text-grayA-9 text-[11px]">
          of {allocated.value} {allocated.unit}
        </span>
      )}
    </div>
  );
}

type MemoryUsageValueProps = {
  usedBytes: number;
  allocatedBytes: number;
  percent: string | null;
};

// Memory: show actual used value + percent in parentheses. "43 MiB (17%)".
// Denominator is context, not worth the visual weight.
function MemoryUsageValue({ usedBytes, allocatedBytes, percent }: MemoryUsageValueProps) {
  const used = formatMemoryParts(bytesToMib(usedBytes));
  return (
    <div className="flex items-baseline gap-1.5 text-[13px] tabular-nums">
      <span>
        <span className="text-gray-12 font-medium">{used.value}</span>
        <span className="font-normal text-grayA-10"> {used.unit}</span>
      </span>
      {percent && allocatedBytes > 0 && (
        <span className="text-grayA-9 text-[11px]">({percent})</span>
      )}
    </div>
  );
}

type DiskUsageValueProps = {
  usedBytes: number;
  allocatedBytes: number;
  percent: string | null;
};

// Disk: same treatment as memory: "212 MiB (4%)". Denominator shown as
// a parenthetical percent because users care about "am I close to full?",
// not the raw allocation (which is static config they already set).
function DiskUsageValue({ usedBytes, allocatedBytes, percent }: DiskUsageValueProps) {
  const used = formatStorageParts(bytesToMib(usedBytes));
  return (
    <div className="flex items-baseline gap-1.5 text-[13px] tabular-nums">
      <span>
        <span className="text-gray-12 font-medium">{used.value}</span>
        {used.unit && <span className="font-normal text-grayA-10"> {used.unit}</span>}
      </span>
      {percent && allocatedBytes > 0 && (
        <span className="text-grayA-9 text-[11px]">({percent})</span>
      )}
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
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <ArrowOppositeDirectionY iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Network</span>
        <div className="ml-auto flex items-center gap-4">
          <NetworkStat label="Egress" color={EGRESS_COLOR} {...egress} />
          <NetworkStat label="Ingress" color={INGRESS_COLOR} {...ingress} />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{
          network_ingress: { label: "Ingress", color: INGRESS_COLOR },
          network_egress: { label: "Egress", color: EGRESS_COLOR },
        }}
        height={140}
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
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <Grid iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Active instances</span>
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
        height={140}
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
};

// CpuSection renders the cpu-millicores timeseries as an area chart so it
// reads the same as disk/network. The tooltip primary value is utilization
// % when allocation is known; raw millicores ride along in the muted hint
// slot so the user can still see the absolute number on hover.
function CpuSection({
  points,
  usedMilli,
  allocatedMilli,
  isLoading,
  isError,
  showDateInTooltip,
  xAxisDomain,
}: CpuSectionProps) {
  const data: AreaChartPoint[] = (points ?? []).map((p) => ({
    originalTimestamp: p.x,
    cpu_usage: p.y,
  }));
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <Bolt iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">CPU usage</span>
        <div className="ml-auto">
          <CpuUsageValue
            usedMilli={usedMilli}
            allocatedMilli={allocatedMilli}
            percent={percent(usedMilli, allocatedMilli)}
          />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ cpu_usage: { label: "CPU", color: CPU_COLOR } }}
        height={140}
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
        // Millicores: 1 vCPU = 1000m. 100m baseline keeps the range
        // reasonable for idle pods while letting spikes register.
        axisFloor={100}
        // Y-axis hidden on CPU — "34m" (millicores) reads as confusing
        // clutter for most users. Tooltip shows a friendlier "%" + "(Nm)".
        hideYAxis
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
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <Focus iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Memory usage</span>
        <div className="ml-auto">
          <MemoryUsageValue
            usedBytes={usedBytes}
            allocatedBytes={allocatedBytes}
            percent={percent(usedBytes, allocatedBytes)}
          />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ memory_usage: { label: "Memory", color: MEMORY_COLOR } }}
        height={140}
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
  const pct = percent(usedBytes, allocatedBytes);
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <Harddrive iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Disk usage</span>
        <div className="ml-auto">
          <DiskUsageValue usedBytes={usedBytes} allocatedBytes={allocatedBytes} percent={pct} />
        </div>
      </div>
      <AreaTimeseriesChart
        data={data}
        config={{ disk_usage: { label: "Disk", color: DISK_COLOR } }}
        height={140}
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

// Compact per-direction stat: "Egress · 12.3 MiB/s · 450 MiB total".
// Peak is the headline (idle pods often show 0 current rate; peak actually
// answers "how bursty was this pod?"). Total is labeled explicitly so the
// eye doesn't pair the bare "450 MiB" with the rate and read them as two
// values of the same metric — they're different quantities (rate × window
// = total) and the `total` suffix keeps that obvious.
function NetworkStat({
  label,
  color,
  peak,
  total,
}: {
  label: string;
  color: string;
  peak: number;
  total: number;
}) {
  const peakParts = formatBytesPerSecondParts(peak);
  const totalParts = formatNetworkBytesParts(total);
  return (
    <div className="flex items-baseline gap-1.5 text-[12px] tabular-nums">
      <span
        className="inline-block rounded-[2px] size-2 translate-y-[1px]"
        style={{ backgroundColor: color }}
        aria-hidden
      />
      <span className="text-grayA-11">{label}</span>
      <span className="text-gray-12 font-medium">{peakParts.value}</span>
      {peakParts.unit && <span className="font-normal text-grayA-10">{peakParts.unit}</span>}
      {total > 0 && (
        <span className="text-grayA-9 text-[11px] whitespace-nowrap">
          · {totalParts.value}
          {totalParts.unit && ` ${totalParts.unit}`} total
        </span>
      )}
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
