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
import { ArrowUpRight, Bolt, ChevronExpandY, Focus, Grid, Harddrive } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { useState } from "react";
import { LogsTimeseriesBarChart } from "./chart";
import { NetworkAreaChart, type NetworkChartPoint } from "./chart/network-area-chart";

// Dashboard panel refreshes every 3s. The chart's live tip reads from raw
// FINAL so new checkpoints (10s heimdall cadence) show up at most one
// refresh after they land. Each refresh fires 4 queries (cpu, memory,
// instances, summary), each scanning at most ~1 minute of raw data per
// workspace+resource. ~1.3 q/s per open panel, well within budget.
const REFETCH_INTERVAL_MS = 3_000;
// Chart height: 48px reads as a sparkline (too small to trust trends),
// 80px gives room for the shape of usage to be read at a glance without
// dominating the panel.
const CHART_HEIGHT = 80;

const WINDOW_LABELS: Record<TimeWindow, string> = {
  "15m": "Past 15 minutes",
  "1h": "Past hour",
  "3h": "Past 3 hours",
  "6h": "Past 6 hours",
  "12h": "Past 12 hours",
  "1d": "Past day",
  "1w": "Past week",
};
const WINDOW_OPTIONS: TimeWindow[] = ["15m", "1h", "3h", "6h", "12h", "1d", "1w"];

type ChartDatum = { originalTimestamp: number; [key: string]: number };

type ChartRowProps = {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  data: ChartDatum[];
  dataKey: string;
  color: string;
  isLoading: boolean;
  valueFormatter?: (value: number) => string;
  showDateInTooltip?: boolean;
};

function ChartRow({
  icon,
  label,
  value,
  data,
  dataKey,
  color,
  isLoading,
  valueFormatter,
  showDateInTooltip,
}: ChartRowProps) {
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex gap-3 items-center">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          {icon}
        </div>
        <span className="text-gray-11 text-xs">{label}</span>
        <div className="ml-auto">{value}</div>
      </div>
      <LogsTimeseriesBarChart
        data={data}
        config={{ [dataKey]: { label, color } }}
        height={CHART_HEIGHT}
        isLoading={isLoading}
        isError={false}
        valueFormatter={valueFormatter}
        showDateInTooltip={showDateInTooltip}
      />
    </div>
  );
}

// Windows that span across days: tooltip needs the date or "4:14 AM" is
// ambiguous. 12h doesn't technically span a day, but a user hovering at
// 6 AM can't tell if they're looking at today or yesterday without the date.
const WINDOWS_NEEDING_DATE: TimeWindow[] = ["12h", "1d", "1w"];

function toChartData(
  points: Array<{ x: number; y: number }> | undefined,
  dataKey: string,
): ChartDatum[] {
  if (!points?.length) {
    return [];
  }
  return points.map((p) => ({ originalTimestamp: p.x, [dataKey]: p.y }));
}

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

  const cpu = trpc.deploy.metrics.getDeploymentCpuTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS },
  );
  const memory = trpc.deploy.metrics.getDeploymentMemoryTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS },
  );
  const disk = trpc.deploy.metrics.getDeploymentDiskTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, enabled: diskEnabled },
  );
  const networkEgress = trpc.deploy.metrics.getDeploymentNetworkEgressTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS },
  );
  const networkIngress = trpc.deploy.metrics.getDeploymentNetworkIngressTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS },
  );
  const instances = trpc.deploy.metrics.getDeploymentInstanceCountTimeseries.useQuery(
    { ...params, window },
    { refetchInterval: REFETCH_INTERVAL_MS, enabled: !isInstanceScoped },
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
        <ChartRow
          icon={<Grid iconSize="sm-regular" className="shrink-0" />}
          label="Active instances"
          value={
            <span className="text-gray-12 font-medium text-[13px] tabular-nums">
              {instanceCount}
              <span className="font-normal text-grayA-10"> vm</span>
            </span>
          }
          data={toChartData(
            instances.data as Array<{ x: number; y: number }> | undefined,
            "active_instances",
          )}
          dataKey="active_instances"
          color="hsl(var(--error-8))"
          isLoading={instances.isLoading}
          // Passing any valueFormatter triggers the compact-label tooltip path.
          valueFormatter={(count) => `${Math.round(count)} vm`}
          showDateInTooltip={showDateInTooltip}
        />
      )}

      <ChartRow
        icon={<Bolt iconSize="sm-regular" className="shrink-0" />}
        label="CPU usage"
        value={
          <CpuUsageValue
            usedMilli={cpuUsedMilli}
            allocatedMilli={cpuAllocatedMilli}
            percent={percent(cpuUsedMilli, cpuAllocatedMilli)}
          />
        }
        data={toChartData(cpu.data as Array<{ x: number; y: number }> | undefined, "cpu_usage")}
        dataKey="cpu_usage"
        color="hsl(var(--feature-8))"
        isLoading={cpu.isLoading}
        // Chart y-value is millicores; tooltip shows utilization % only.
        // No % fallback ("—") when allocation is unknown so we don't
        // silently claim 0%.
        valueFormatter={(millicores) =>
          cpuAllocatedMilli > 0
            ? formatTooltipPercent((millicores / cpuAllocatedMilli) * 100)
            : `${Math.round(millicores)}m`
        }
        showDateInTooltip={showDateInTooltip}
      />

      <ChartRow
        icon={<Focus iconSize="sm-regular" className="shrink-0" />}
        label="Memory usage"
        value={
          <MemoryUsageValue
            usedBytes={memUsedBytes}
            allocatedBytes={memAllocatedBytes}
            percent={percent(memUsedBytes, memAllocatedBytes)}
          />
        }
        data={toChartData(
          memory.data as Array<{ x: number; y: number }> | undefined,
          "memory_usage",
        )}
        dataKey="memory_usage"
        color="hsl(var(--info-8))"
        isLoading={memory.isLoading}
        // Chart y-value is bytes; tooltip shows "43 MiB (17%)".
        valueFormatter={(bytes) => {
          const parts = formatMemoryParts(bytesToMib(bytes));
          const body = `${parts.value} ${parts.unit}`;
          if (memAllocatedBytes <= 0) {
            return body;
          }
          return `${body} (${formatTooltipPercent((bytes / memAllocatedBytes) * 100)})`;
        }}
        showDateInTooltip={showDateInTooltip}
      />

      {diskEnabled && (
        <ChartRow
          icon={<Harddrive iconSize="sm-regular" className="shrink-0" />}
          label="Disk usage"
          value={
            <DiskUsageValue
              usedBytes={diskUsedBytes}
              allocatedBytes={diskAllocatedBytes}
              percent={percent(diskUsedBytes, diskAllocatedBytes)}
            />
          }
          data={toChartData(disk.data as Array<{ x: number; y: number }> | undefined, "disk_usage")}
          dataKey="disk_usage"
          color="hsl(var(--warning-8))"
          isLoading={disk.isLoading}
          valueFormatter={(bytes) => {
            const parts = formatMemoryParts(bytesToMib(bytes));
            const body = `${parts.value} ${parts.unit}`;
            if (diskAllocatedBytes <= 0) {
              return body;
            }
            return `${body} (${formatTooltipPercent((bytes / diskAllocatedBytes) * 100)})`;
          }}
          showDateInTooltip={showDateInTooltip}
        />
      )}

      <NetworkSection
        egressPoints={networkEgress.data as Array<{ x: number; y: number }> | undefined}
        ingressPoints={networkIngress.data as Array<{ x: number; y: number }> | undefined}
        isLoading={networkEgress.isLoading || networkIngress.isLoading}
        showDateInTooltip={showDateInTooltip}
      />
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
  showDateInTooltip?: boolean;
};

const EGRESS_COLOR = "hsl(var(--error-8))";
const INGRESS_COLOR = "hsl(var(--success-8))";

function NetworkSection({
  egressPoints,
  ingressPoints,
  isLoading,
  showDateInTooltip,
}: NetworkSectionProps) {
  const egress = summarizeRateSeries(egressPoints);
  const ingress = summarizeRateSeries(ingressPoints);
  const data = mergeNetworkSeries(egressPoints, ingressPoints);
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          <ArrowUpRight iconSize="sm-regular" className="shrink-0" />
        </div>
        <span className="text-gray-11 text-xs">Network</span>
        <div className="ml-auto flex items-center gap-4">
          <NetworkStat label="Egress" color={EGRESS_COLOR} {...egress} />
          <NetworkStat label="Ingress" color={INGRESS_COLOR} {...ingress} />
        </div>
      </div>
      <NetworkAreaChart
        data={data}
        config={{
          network_ingress: { label: "Ingress", color: INGRESS_COLOR },
          network_egress: { label: "Egress", color: EGRESS_COLOR },
        }}
        height={140}
        isLoading={isLoading}
        showDateInTooltip={showDateInTooltip}
      />
    </div>
  );
}

// Compact per-direction stat: "Egress · 12.3 MiB/s peak · 450 MiB total".
// Peak is the headline (idle pods often show 0 current rate; peak actually
// answers "how bursty was this pod?"). Total is the context.
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
        <span className="text-grayA-9 text-[11px]">
          · {totalParts.value}
          {totalParts.unit && ` ${totalParts.unit}`}
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
): NetworkChartPoint[] {
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
function summarizeRateSeries(points: Array<{ x: number; y: number }> | undefined): {
  peak: number;
  total: number;
} {
  if (!points?.length) {
    return { peak: 0, total: 0 };
  }
  let peak = 0;
  let total = 0;
  for (let i = 0; i < points.length; i++) {
    const y = points[i].y;
    if (y > peak) {
      peak = y;
    }
    if (i > 0) {
      const dtSec = (points[i].x - points[i - 1].x) / 1000;
      total += points[i - 1].y * dtSec;
    }
  }
  return { peak, total };
}
