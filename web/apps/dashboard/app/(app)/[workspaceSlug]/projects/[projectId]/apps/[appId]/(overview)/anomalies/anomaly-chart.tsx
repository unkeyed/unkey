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
import { Empty, Skeleton } from "@unkey/ui";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  type LabelProps,
  Line,
  ReferenceArea,
  ReferenceDot,
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

type HoveredAnnotation = {
  alerts: AlertListItem[];
  anchorX: number;
  anchorY: number;
  kind: "dot" | "window";
};

const chartHeightPx = 380;
const plotTopPx = 8;
const plotRightPx = 16;
const plotBottomPx = 38;
const plotLeftPx = 80;

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
  const alertDots = useMemo(
    () =>
      alerts.flatMap((alert) => {
        if (seriesMetricForAlert(alert.metric) !== metric) {
          return [];
        }
        const point = extremePointInAlertWindow(alert, buckets);
        return point ? [{ alert, point }] : [];
      }),
    [alerts, buckets, metric],
  );
  const [chartWidthPx, setChartWidthPx] = useState(1_000);
  const [hoveredAnnotation, setHoveredAnnotation] = useState<HoveredAnnotation | null>(null);

  useEffect(() => {
    const container = chartAreaRef.current;
    if (!container) {
      return;
    }
    const updateWidth = () => setChartWidthPx(container.clientWidth);
    updateWidth();
    const observer = new ResizeObserver(updateWidth);
    observer.observe(container);
    return () => observer.disconnect();
  }, []);

  const deploymentLabels = useMemo(
    () =>
      data
        ? groupDeploymentLabels(deployments, data.startMs, data.endMs, chartWidthPx)
        : new Map<string, string>(),
    [chartWidthPx, data, deployments],
  );

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
  const showAnnotationTooltip = (
    annotationAlerts: AlertListItem[],
    element: SVGElement,
    kind: HoveredAnnotation["kind"],
  ) => {
    const chartBounds = chartAreaRef.current?.getBoundingClientRect();
    if (!chartBounds) {
      return;
    }
    const elementBounds = element.getBoundingClientRect();
    setHoveredAnnotation({
      alerts: annotationAlerts,
      anchorX: elementBounds.left + elementBounds.width / 2 - chartBounds.left,
      anchorY:
        (kind === "window" ? elementBounds.top : elementBounds.top + elementBounds.height / 2) -
        chartBounds.top,
      kind,
    });
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
        <ChartContainer
          config={chartConfig}
          className="h-[380px] w-full aspect-auto"
          aria-label={`${alertSeriesMetricLabel(metric)} anomaly chart`}
        >
          <ComposedChart
            data={buckets}
            accessibilityLayer
            margin={{ top: plotTopPx, right: plotRightPx, bottom: 8, left: 8 }}
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
              activeDot={false}
              tooltipType="none"
              isAnimationActive={false}
            />
            {alerts.map((alert) => {
              const selectedMetric = seriesMetricForAlert(alert.metric) === metric;
              const windowAlerts = alertsInWindow(alerts, alert);
              return (
                <ReferenceArea
                  key={alert.id}
                  x1={alert.windowStart}
                  x2={alert.windowEnd}
                  fill="hsl(var(--error-9))"
                  fillOpacity={alert.id === selectedAlertId ? 0.22 : 0.1}
                  stroke="none"
                  label={selectedMetric ? false : <AnomalyTopTick />}
                  style={{ pointerEvents: selectedMetric ? "none" : "auto", cursor: "help" }}
                  onMouseEnter={(event) =>
                    showAnnotationTooltip(windowAlerts, event.currentTarget, "window")
                  }
                  onMouseMove={(event) =>
                    showAnnotationTooltip(windowAlerts, event.currentTarget, "window")
                  }
                  onMouseLeave={() => setHoveredAnnotation(null)}
                />
              );
            })}
            {showDeployments
              ? deployments.map((deployment) => {
                  const label = deploymentLabels.get(deployment.id);
                  return (
                    <ReferenceLine
                      key={deployment.id}
                      x={deployment.createdAt}
                      stroke="hsl(var(--gray-9))"
                      strokeDasharray="4 4"
                      strokeOpacity={0.7}
                      label={
                        label ? (
                          <DeploymentTagLabel label={label} chartWidthPx={chartWidthPx} />
                        ) : (
                          false
                        )
                      }
                      style={{ pointerEvents: "none" }}
                    />
                  );
                })
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
                {metric === "requests" ? (
                  <Line
                    type="monotone"
                    dataKey="lowerBound"
                    stroke="var(--color-expectedMean)"
                    strokeWidth={1}
                    strokeDasharray="2 4"
                    strokeOpacity={0.7}
                    dot={false}
                    activeDot={false}
                    tooltipType="none"
                    isAnimationActive={false}
                  />
                ) : null}
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke="var(--color-value)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={false}
                  isAnimationActive={false}
                />
              </>
            )}
            {alertDots.map(({ alert, point }) => (
              <ReferenceDot
                key={alert.id}
                x={point.time}
                y={point.value}
                r={4}
                fill="hsl(var(--error-9))"
                stroke="white"
                strokeWidth={2}
                shape={
                  <AlertDotShape
                    label={`${alertMetricLabel(alert.metric)} anomaly`}
                    onHover={(element) =>
                      showAnnotationTooltip(alertsInWindow(alerts, alert), element, "dot")
                    }
                    onLeave={() => setHoveredAnnotation(null)}
                  />
                }
                label={
                  alert.id === selectedAlertId ? (
                    <FocusedAlertDotLabel
                      label={alertMetricLabel(alert.metric)}
                      chartWidthPx={chartWidthPx}
                    />
                  ) : (
                    false
                  )
                }
              />
            ))}
          </ComposedChart>
        </ChartContainer>
        {hoveredAnnotation ? (
          <AnomalyWindowTooltip hovered={hoveredAnnotation} chartWidthPx={chartWidthPx} />
        ) : null}
      </div>
    </div>
  );
}

function extremePointInAlertWindow(
  alert: AlertListItem,
  buckets: Array<{ time: number; value: number }>,
): { time: number; value: number } | null {
  const points = buckets.filter(
    (point) => point.time >= alert.windowStart && point.time < alert.windowEnd,
  );
  return (
    points.reduce<(typeof points)[number] | null>((selected, point) => {
      if (!selected) {
        return point;
      }
      const isMoreExtreme =
        alert.metric === "requests_drop"
          ? point.value < selected.value
          : point.value > selected.value;
      return isMoreExtreme ? point : selected;
    }, null) ?? null
  );
}

function AnomalyTopTick({ viewBox }: { viewBox?: LabelProps["viewBox"] }) {
  const box = cartesianLabelBox(viewBox);
  if (!box) {
    return null;
  }
  return (
    <rect
      x={box.x}
      y={box.y}
      width={Math.max(box.width, 3)}
      height={3}
      fill="hsl(var(--error-9))"
      pointerEvents="none"
    />
  );
}

function AlertDotShape({
  cx,
  cy,
  label,
  onHover,
  onLeave,
}: {
  cx?: number;
  cy?: number;
  label: string;
  onHover: (element: SVGCircleElement) => void;
  onLeave: () => void;
}) {
  if (cx === undefined || cy === undefined) {
    return <circle />;
  }
  return (
    <circle
      cx={cx}
      cy={cy}
      r={4}
      fill="hsl(var(--error-9))"
      stroke="white"
      strokeWidth={2}
      aria-label={label}
      className="cursor-help"
      onMouseEnter={(event) => onHover(event.currentTarget)}
      onMouseMove={(event) => onHover(event.currentTarget)}
      onMouseLeave={onLeave}
    />
  );
}

function FocusedAlertDotLabel({
  label,
  chartWidthPx,
  viewBox,
}: {
  label: string;
  chartWidthPx: number;
  viewBox?: LabelProps["viewBox"];
}) {
  const box = cartesianLabelBox(viewBox);
  if (!box) {
    return null;
  }
  const width = estimateTagWidth(label);
  const centerX = clamp(
    box.x + box.width / 2,
    plotLeftPx + width / 2,
    chartWidthPx - plotRightPx - width / 2,
  );
  const y = Math.max(plotTopPx + 2, box.y - 24);
  return (
    <g pointerEvents="none">
      <rect
        x={centerX - width / 2}
        y={y}
        width={width}
        height={18}
        rx={9}
        fill="hsl(var(--error-3))"
        stroke="hsl(var(--error-7))"
      />
      <text
        x={centerX}
        y={y + 12}
        textAnchor="middle"
        fontSize={9}
        fontWeight={500}
        fill="hsl(var(--error-11))"
      >
        {label}
      </text>
    </g>
  );
}

function DeploymentTagLabel({
  label,
  chartWidthPx,
  viewBox,
}: {
  label: string;
  chartWidthPx: number;
  viewBox?: LabelProps["viewBox"];
}) {
  const box = cartesianLabelBox(viewBox);
  if (!box) {
    return null;
  }
  const width = estimateTagWidth(label);
  const centerX = clamp(
    box.x + box.width / 2,
    plotLeftPx + width / 2,
    chartWidthPx - plotRightPx - width / 2,
  );
  return (
    <g pointerEvents="none">
      <rect
        x={centerX - width / 2}
        y={box.y + 3}
        width={width}
        height={18}
        rx={9}
        fill="hsl(var(--gray-2))"
        stroke="hsl(var(--gray-6))"
      />
      <text
        x={centerX}
        y={box.y + 15}
        textAnchor="middle"
        fontFamily="var(--font-mono)"
        fontSize={9}
        fill="hsl(var(--gray-11))"
      >
        {label}
      </text>
    </g>
  );
}

function groupDeploymentLabels(
  deployments: DeploymentMarker[],
  startMs: number,
  endMs: number,
  chartWidthPx: number,
): Map<string, string> {
  const plotWidth = Math.max(chartWidthPx - plotLeftPx - plotRightPx, 1);
  const groups: Array<{ deployment: DeploymentMarker; hidden: number; x: number; width: number }> =
    [];
  for (const deployment of [...deployments].sort((a, b) => a.createdAt - b.createdAt)) {
    const x = plotLeftPx + ((deployment.createdAt - startMs) / (endMs - startMs)) * plotWidth;
    let hidden = 0;
    let label = deployment.label;
    let width = estimateTagWidth(label);
    let previous = groups.at(-1);
    while (previous && x - previous.x < (width + previous.width) / 2 + 4) {
      hidden += previous.hidden + 1;
      groups.pop();
      label = `${deployment.label} +${hidden}`;
      width = estimateTagWidth(label);
      previous = groups.at(-1);
    }
    groups.push({ deployment, hidden, x, width });
  }
  return new Map(
    groups.map(({ deployment, hidden }) => [
      deployment.id,
      hidden > 0 ? `${deployment.label} +${hidden}` : deployment.label,
    ]),
  );
}

function cartesianLabelBox(viewBox: LabelProps["viewBox"] | undefined) {
  return viewBox && "x" in viewBox ? viewBox : null;
}

function estimateTagWidth(label: string): number {
  return label.length * 5.5 + 16;
}

function alertsInWindow(alerts: AlertListItem[], hovered: AlertListItem): AlertListItem[] {
  return alerts.filter(
    (alert) => alert.windowStart < hovered.windowEnd && alert.windowEnd > hovered.windowStart,
  );
}

function AnomalyWindowTooltip({
  hovered,
  chartWidthPx,
}: {
  hovered: HoveredAnnotation;
  chartWidthPx: number;
}) {
  const width = 256;
  const estimatedHeight = 44 + hovered.alerts.length * 54;
  const left = clamp(
    hovered.anchorX,
    plotLeftPx + width / 2,
    chartWidthPx - plotRightPx - width / 2,
  );
  const above = hovered.anchorY - estimatedHeight - 10;
  const top =
    hovered.kind === "dot" && above >= plotTopPx
      ? above
      : Math.min(hovered.anchorY + 10, chartHeightPx - plotBottomPx - estimatedHeight);
  return (
    <div
      className="pointer-events-none absolute z-30 w-64 -translate-x-1/2 overflow-hidden rounded-lg border border-grayA-5 bg-gray-2 shadow-lg"
      style={{ left, top: Math.max(plotTopPx, top) }}
    >
      <div className="border-b border-grayA-4 px-3 py-2 text-xs font-medium text-gray-12">
        {hovered.alerts.length === 1 ? "Anomaly in this window" : "Anomalies in this window"}
      </div>
      <div className="divide-y divide-grayA-4">
        {hovered.alerts.map((alert) => (
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
  if (alert.metric === "requests_drop") {
    return `${formatAlertValue(alert.metric, alert.observedValue)} vs ${formatAlertValue(
      alert.metric,
      alert.baselineMean,
    )} recent (${formatAlertDistance(
      alert.metric,
      alert.observedValue,
      alert.baselineMean,
      alert.baselineStddev,
    )})`;
  }
  return `${formatAlertValue(alert.metric, alert.observedValue)} vs ${formatAlertValue(
    alert.metric,
    alert.baselineMean,
  )} expected`;
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.max(minimum, Math.min(value, maximum));
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
