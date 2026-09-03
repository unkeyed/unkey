"use client";

import {
  alertMetricLabel,
  alertSeriesMetricLabel,
  formatAlertDistance,
  formatAlertSeriesAxisValue,
  formatAlertSeriesValue,
  formatAlertValue,
  hasFixedAlertThreshold,
  seriesMetricForAlert,
} from "@/components/alerts/format";
import { AlertStatusBadge } from "@/components/alerts/status-badge";
import type { AlertListItem, AlertSeriesData, AlertSeriesMetric } from "@/components/alerts/types";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { cn } from "@/lib/utils";
import { Empty, Skeleton } from "@unkey/ui";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceArea,
  ReferenceLine,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { fiveMinutesMs } from "./range";

const chartConfig = {
  value: { label: "Observed", color: "hsl(var(--info-9))" },
  expectedMean: { label: "Expected", color: "hsl(var(--gray-10))" },
  expectedBand: { label: "Expected range", color: "hsl(var(--info-5))" },
} satisfies ChartConfig;

type DeploymentMarker = {
  id: string;
  createdAt: number;
  gitSha: string | null;
  label: string;
};

export function AnomalyChart({
  metric,
  data,
  alerts,
  deployments,
  selectedAlertId,
  showDeployments,
  loading,
  error,
  onZoom,
}: {
  metric: AlertSeriesMetric;
  data: AlertSeriesData | undefined;
  alerts: AlertListItem[];
  deployments: DeploymentMarker[];
  selectedAlertId: string | null;
  showDeployments: boolean;
  loading: boolean;
  error: boolean;
  onZoom: (range: { startMs: number; endMs: number }) => void;
}) {
  const chartAreaRef = useRef<HTMLDivElement>(null);
  const dragStart = useRef<number | null>(null);
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const buckets = useMemo(
    () =>
      (data?.buckets ?? []).map((point) => ({
        ...point,
        expectedBand:
          (metric === "requests" ? point.lowerBound : point.expectedMean) === null ||
          point.upperBound === null
            ? null
            : [metric === "requests" ? point.lowerBound : point.expectedMean, point.upperBound],
      })),
    [data?.buckets, metric],
  );
  const [hoveredAlerts, setHoveredAlerts] = useState<AlertListItem[] | null>(null);

  if (loading) {
    return <Skeleton className="h-[430px] w-full rounded-lg" />;
  }
  if (error) {
    return (
      <Empty className="h-[430px] w-full rounded-lg border border-grayA-4">
        <Empty.Title>Chart unavailable</Empty.Title>
        <Empty.Description>We could not load telemetry for this range.</Empty.Description>
      </Empty>
    );
  }
  if (!data || buckets.length === 0) {
    return (
      <Empty className="h-[430px] w-full rounded-lg border border-grayA-4">
        <Empty.Title>No telemetry</Empty.Title>
        <Empty.Description>No closed metric buckets exist in this range.</Empty.Description>
      </Empty>
    );
  }

  const selectIndex = (event: React.MouseEvent): number | null => {
    const element = chartAreaRef.current;
    if (!element) {
      return null;
    }
    const bounds = element.getBoundingClientRect();
    const x = Math.max(0, Math.min(event.clientX - bounds.left - 72, bounds.width - 96));
    const plotWidth = Math.max(bounds.width - 96, 1);
    return Math.min(Math.floor((x / plotWidth) * buckets.length), buckets.length - 1);
  };
  const handleMouseDown = (event: React.MouseEvent) => {
    const index = selectIndex(event);
    if (index === null) {
      return;
    }
    dragStart.current = index;
    setSelection({ start: index, end: index });
  };
  const handleMouseMove = (event: React.MouseEvent) => {
    if (dragStart.current === null) {
      return;
    }
    const index = selectIndex(event);
    if (index !== null) {
      setSelection({ start: dragStart.current, end: index });
    }
  };
  const handleMouseUp = () => {
    const current = selection;
    dragStart.current = null;
    setSelection(null);
    if (!current || Math.abs(current.end - current.start) < 2) {
      return;
    }
    const first = Math.min(current.start, current.end);
    const last = Math.max(current.start, current.end);
    const startPoint = buckets[first];
    const endPoint = buckets[last];
    if (startPoint && endPoint) {
      onZoom({ startMs: startPoint.time, endMs: endPoint.time + fiveMinutesMs });
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border border-grayA-4 bg-gray-1">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-grayA-4 px-5 py-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-12">{alertSeriesMetricLabel(metric)}</h2>
          <p className="text-xs text-gray-9">Five-minute production buckets · drag to zoom</p>
        </div>
        <div className="flex flex-wrap items-center gap-4 text-xs text-gray-10">
          <ChartLegend color="bg-info-9" label="Observed" />
          {metric === "health" ? null : metric === "memory_utilization" ? (
            <ChartLegend color="border-t-2 border-dashed border-warning-9" label="Limit" />
          ) : (
            <>
              <ChartLegend color="border-t-2 border-dashed border-gray-10" label="Expected" />
              <ChartLegend color="bg-infoA-3" label="Expected range" />
            </>
          )}
          <ChartLegend color="bg-errorA-4" label="Anomaly" />
        </div>
      </div>
      <div
        ref={chartAreaRef}
        className="relative cursor-crosshair select-none"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={() => {
          dragStart.current = null;
          setSelection(null);
        }}
      >
        <ChartAnnotations
          metric={metric}
          alerts={alerts}
          deployments={showDeployments ? deployments : []}
          selectedAlertId={selectedAlertId}
          startMs={data.startMs}
          endMs={data.endMs}
        />
        <ChartContainer
          config={chartConfig}
          className="h-[380px] w-full aspect-auto px-3 pt-4 pb-2"
          aria-label={`${alertSeriesMetricLabel(metric)} anomaly chart`}
        >
          <ComposedChart
            data={buckets}
            accessibilityLayer
            margin={{ top: 104, right: 16, bottom: 8, left: 8 }}
          >
            <CartesianGrid
              vertical={false}
              stroke="hsl(var(--gray-6))"
              strokeDasharray="3 3"
              strokeOpacity={0.45}
            />
            <XAxis
              dataKey="time"
              type="number"
              scale="time"
              domain={["dataMin", "dataMax"]}
              tickLine={false}
              axisLine={false}
              minTickGap={48}
              tickFormatter={(value) => formatChartTime(Number(value), data.endMs - data.startMs)}
            />
            <YAxis
              width={72}
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => formatAlertSeriesAxisValue(metric, Number(value))}
            />
            <Tooltip
              cursor={{ stroke: "hsl(var(--gray-8))", strokeDasharray: "4 4" }}
              contentStyle={{
                border: "1px solid hsl(var(--gray-6))",
                borderRadius: 8,
                background: "hsl(var(--gray-2))",
                color: "hsl(var(--gray-12))",
                fontSize: 12,
              }}
              labelFormatter={(value) => formatChartTooltipTime(Number(value))}
              formatter={(value, name) => [
                formatAlertSeriesValue(metric, Number(value ?? 0)),
                name === "expectedMean" ? "Expected" : "Observed",
              ]}
            />
            <Area
              dataKey="expectedBand"
              fill="hsl(var(--info-9))"
              fillOpacity={0.1}
              stroke="none"
              tooltipType="none"
              isAnimationActive={false}
            />
            {alerts.map((alert) => (
              <ReferenceArea
                key={alert.id}
                x1={alert.windowStart}
                x2={alert.windowEnd}
                fill="hsl(var(--error-9))"
                fillOpacity={alert.id === selectedAlertId ? 0.28 : 0.14}
                stroke="hsl(var(--error-8))"
                strokeOpacity={alert.id === selectedAlertId ? 1 : 0.45}
              />
            ))}
            {showDeployments
              ? deployments.map((deployment) => (
                  <ReferenceLine
                    key={deployment.id}
                    x={deployment.createdAt}
                    stroke="hsl(var(--gray-9))"
                    strokeDasharray="4 4"
                    strokeOpacity={0.7}
                  />
                ))
              : null}
            {selection ? (
              <ReferenceArea
                x1={buckets[Math.min(selection.start, selection.end)]?.time}
                x2={buckets[Math.max(selection.start, selection.end)]?.time}
                fill="hsl(var(--info-9))"
                fillOpacity={0.16}
                stroke="hsl(var(--info-8))"
              />
            ) : null}
            {metric === "memory_utilization" ? (
              <ReferenceLine
                y={0.9}
                stroke="hsl(var(--warning-9))"
                strokeWidth={1.5}
                strokeDasharray="5 5"
              />
            ) : null}
            {metric === "health" ? (
              <Bar
                dataKey="value"
                fill="var(--color-value)"
                maxBarSize={12}
                radius={[2, 2, 0, 0]}
              />
            ) : (
              <>
                <Line
                  type="monotone"
                  dataKey="expectedMean"
                  stroke="var(--color-expectedMean)"
                  strokeWidth={1.5}
                  strokeDasharray="5 5"
                  dot={false}
                  activeDot={false}
                  isAnimationActive={false}
                />
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke="var(--color-value)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, fill: "hsl(var(--info-9))" }}
                  isAnimationActive={false}
                />
              </>
            )}
          </ComposedChart>
        </ChartContainer>
        <AnomalyHoverAreas
          alerts={alerts}
          startMs={data.startMs}
          endMs={data.endMs}
          onHover={setHoveredAlerts}
        />
        {hoveredAlerts ? <AnomalyWindowTooltip alerts={hoveredAlerts} /> : null}
      </div>
    </div>
  );
}

function ChartAnnotations({
  metric,
  alerts,
  deployments,
  selectedAlertId,
  startMs,
  endMs,
}: {
  metric: AlertSeriesMetric;
  alerts: AlertListItem[];
  deployments: DeploymentMarker[];
  selectedAlertId: string | null;
  startMs: number;
  endMs: number;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [widthPx, setWidthPx] = useState(1_000);
  useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }
    const updateWidth = () => setWidthPx(container.clientWidth);
    updateWidth();
    const observer = new ResizeObserver(updateWidth);
    observer.observe(container);
    return () => observer.disconnect();
  }, []);

  const position = (time: number) =>
    Math.max(3, Math.min(97, ((time - startMs) / (endMs - startMs)) * 100));
  const labeledAlerts = alerts.filter(
    (alert) => seriesMetricForAlert(alert.metric) === metric || alert.id === selectedAlertId,
  );
  const markerGroups = groupAlertMarkers(
    alerts.filter((alert) => !labeledAlerts.includes(alert)),
    startMs,
    endMs,
    widthPx,
  );

  return (
    <div
      ref={containerRef}
      className="pointer-events-none absolute inset-x-0 top-0 z-10 h-[116px] overflow-hidden"
    >
      {deployments.map((deployment) => {
        const left = position(deployment.createdAt);
        return (
          <span
            key={deployment.id}
            className={cn(
              "absolute top-2 rounded-full border border-grayA-5 bg-gray-2 px-1.5 py-0.5 font-mono text-[9px] text-gray-10 shadow-sm",
              annotationAnchor(left),
            )}
            style={{ left: `${left}%` }}
          >
            {deployment.label}
          </span>
        );
      })}
      {labeledAlerts.map((alert, index) => {
        const left = position(alert.windowStart);
        return (
          <span
            key={alert.id}
            className={cn(
              "absolute whitespace-nowrap rounded-full border border-errorA-5 bg-error-3 px-1.5 py-0.5 text-[9px] font-medium text-error-11 shadow-sm",
              annotationAnchor(left),
            )}
            style={{ left: `${left}%`, top: 34 + (index % 2) * 18 }}
          >
            {alert.id === selectedAlertId ? "● " : ""}
            {alertMetricLabel(alert.metric)}
          </span>
        );
      })}
      {markerGroups.map((group) => {
        const firstAlert = group.alerts.at(0);
        if (!firstAlert) {
          return null;
        }
        const left = position(group.time);
        return group.alerts.length === 1 ? (
          <span
            key={firstAlert.id}
            className="absolute top-[82px] size-2 -translate-x-1/2 rounded-full border border-errorA-7 bg-error-9"
            style={{ left: `${left}%` }}
            aria-label={`${alertMetricLabel(firstAlert.metric)} anomaly`}
          />
        ) : (
          <span
            key={group.alerts.map((alert) => alert.id).join(":")}
            className={cn(
              "absolute top-[76px] rounded-full border border-errorA-6 bg-error-3 px-1.5 py-0.5 text-[9px] font-medium text-error-11",
              annotationAnchor(left),
            )}
            style={{ left: `${left}%` }}
            aria-label={`${group.alerts.length} nearby anomalies`}
          >
            +{group.alerts.length}
          </span>
        );
      })}
    </div>
  );
}

function groupAlertMarkers(
  alerts: AlertListItem[],
  startMs: number,
  endMs: number,
  widthPx: number,
): Array<{ time: number; alerts: AlertListItem[] }> {
  const proximityMs = ((endMs - startMs) * 24) / Math.max(widthPx, 1);
  const groups: Array<{ time: number; latestTime: number; alerts: AlertListItem[] }> = [];
  for (const alert of [...alerts].sort((a, b) => a.windowStart - b.windowStart)) {
    const current = groups.at(-1);
    if (current && alert.windowStart - current.latestTime <= proximityMs) {
      current.latestTime = alert.windowStart;
      current.alerts.push(alert);
    } else {
      groups.push({ time: alert.windowStart, latestTime: alert.windowStart, alerts: [alert] });
    }
  }
  return groups;
}

function alertsInWindow(alerts: AlertListItem[], hovered: AlertListItem): AlertListItem[] {
  return alerts.filter(
    (alert) => alert.windowStart < hovered.windowEnd && alert.windowEnd > hovered.windowStart,
  );
}

function AnomalyHoverAreas({
  alerts,
  startMs,
  endMs,
  onHover,
}: {
  alerts: AlertListItem[];
  startMs: number;
  endMs: number;
  onHover: (alerts: AlertListItem[] | null) => void;
}) {
  const position = (time: number) =>
    Math.max(0, Math.min(100, ((time - startMs) / (endMs - startMs)) * 100));
  return (
    <div className="pointer-events-none absolute top-[118px] right-6 bottom-10 left-[84px] z-20">
      {alerts.map((alert) => {
        const left = position(alert.windowStart);
        const width = position(alert.windowEnd) - left;
        return (
          <div
            key={alert.id}
            className="pointer-events-auto absolute inset-y-0 cursor-help"
            style={{ left: `${left}%`, width: `max(8px, ${width}%)` }}
            onMouseEnter={() => onHover(alertsInWindow(alerts, alert))}
            onMouseMove={() => onHover(alertsInWindow(alerts, alert))}
            onMouseLeave={() => onHover(null)}
          />
        );
      })}
    </div>
  );
}

function AnomalyWindowTooltip({ alerts }: { alerts: AlertListItem[] }) {
  return (
    <div className="pointer-events-none absolute right-4 top-[118px] z-30 w-64 overflow-hidden rounded-lg border border-grayA-5 bg-gray-2 shadow-lg">
      <div className="border-b border-grayA-4 px-3 py-2 text-xs font-medium text-gray-12">
        {alerts.length === 1 ? "Anomaly in this window" : "Anomalies in this window"}
      </div>
      <div className="divide-y divide-grayA-4">
        {alerts.map((alert) => (
          <div key={alert.id} className="flex flex-col gap-1.5 px-3 py-2.5">
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-gray-12">
                {alertMetricLabel(alert.metric)}
              </span>
              <AlertStatusBadge status={alert.status} />
            </div>
            <span className="text-xs tabular-nums text-gray-10">
              {formatAlertExpectation(alert)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function formatAlertExpectation(alert: AlertListItem): string {
  if (hasFixedAlertThreshold(alert.metric)) {
    return formatAlertDistance(
      alert.metric,
      alert.observedValue,
      alert.baselineMean,
      alert.baselineStddev,
    );
  }
  return `${formatAlertValue(alert.metric, alert.observedValue)} vs ${formatAlertValue(
    alert.metric,
    alert.baselineMean,
  )} expected`;
}

function annotationAnchor(positionPercent: number): string {
  if (positionPercent < 10) {
    return "translate-x-0";
  }
  if (positionPercent > 90) {
    return "-translate-x-full";
  }
  return "-translate-x-1/2";
}

function ChartLegend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-2 w-4 rounded-sm ${color}`} aria-hidden="true" />
      {label}
    </span>
  );
}

function formatChartTime(value: number, rangeMs: number): string {
  return new Intl.DateTimeFormat("en-US", {
    ...(rangeMs > 24 * 60 * 60 * 1000 ? { month: "short", day: "numeric" } : {}),
    hour: "numeric",
    minute: "2-digit",
  }).format(value);
}

function formatChartTooltipTime(value: number): string {
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(value);
}
