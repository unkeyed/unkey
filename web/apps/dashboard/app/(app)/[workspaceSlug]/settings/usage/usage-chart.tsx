"use client";

import { AreaTimeseriesChart, type ValueParts } from "@/components/charts/area-timeseries";
import { AppEnvironmentFilterList } from "@/components/deploy/app-environment-filter-list";
import { getAppEnvironmentSelection } from "@/components/deploy/app-environment-selection";
import type { ChartConfig } from "@/components/ui/chart";
import { Switch } from "@/components/ui/switch";
import { formatCompactQuantity } from "@/lib/fmt";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import type { DeployUsageTimeseriesGroup, DeployUsageTimeseriesInterval } from "@unkey/clickhouse";
import { ChevronExpandY } from "@unkey/icons";
import {
  ItemSeparator,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from "@unkey/ui";
import { useId, useMemo, useState } from "react";
import type { ComputeTree } from "./compute-tree";
import { type UsageMetric, buildUsageChart } from "./usage-chart-data";

const METRICS: Record<UsageMetric, { label: string; unit: string; axisUnit: string }> = {
  egressGiB: { label: "Public egress", unit: "GiB", axisUnit: "GiB" },
  cpuHours: { label: "CPU", unit: "vCPU-hours", axisUnit: "vCPU-h" },
  memoryGiBHours: { label: "Memory", unit: "GiB-hours", axisUnit: "GiB-h" },
  diskGiBHours: { label: "Storage", unit: "GiB-hours", axisUnit: "GiB-h" },
};

const GROUP_LABELS: Record<DeployUsageTimeseriesGroup, string> = {
  total: "Total",
  project: "Project",
  app: "App",
  environment: "Environment",
};

const INTERVAL_LABELS: Record<DeployUsageTimeseriesInterval, string> = {
  hour: "Hourly",
  day: "Daily",
};

type MonthsAgo = 0 | 1 | 2;
const PERIODS: MonthsAgo[] = [0, 1, 2];
const ALL_PROJECTS = "all-projects";
const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * 60 * 60 * 1000;
const MAX_DEPLOYMENT_LABELS = 6;

type UsageScopeFilter = {
  field: "appId" | "environmentId";
  value: string;
};

function createUsageScopeFilter(field: UsageScopeFilter["field"], value: string): UsageScopeFilter {
  return { field, value };
}

function isUsageMetric(value: string): value is UsageMetric {
  return value in METRICS;
}

function isUsageGroup(value: string): value is DeployUsageTimeseriesGroup {
  return value === "total" || value === "project" || value === "app" || value === "environment";
}

function isUsageInterval(value: string): value is DeployUsageTimeseriesInterval {
  return value === "hour" || value === "day";
}

function isMonthsAgo(value: number): value is MonthsAgo {
  return value === 0 || value === 1 || value === 2;
}

function billingPeriod(now: Date, monthsAgo: MonthsAgo) {
  const start = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo, 1);
  const end =
    monthsAgo === 0
      ? now.getTime()
      : Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo + 1, 1);
  return {
    start,
    end,
    label: new Date(start).toLocaleDateString("en-US", {
      timeZone: "UTC",
      month: "long",
      year: "numeric",
    }),
  };
}

function currentBucketStart(now: Date, interval: DeployUsageTimeseriesInterval): number {
  return Date.UTC(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate(),
    interval === "hour" ? now.getUTCHours() : 0,
  );
}

function daysInPeriod(start: number, end: number): number[] {
  const days: number[] = [];
  for (let day = start; day < end; day += DAY_MS) {
    days.push(day);
  }
  return days;
}

function formatDay(time: number, includeYear = true): string {
  return new Date(time).toLocaleDateString("en-US", {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    ...(includeYear ? { year: "numeric" } : {}),
  });
}

function formatUsageValue(value: number, unit: string): ValueParts {
  const formatted =
    value > 0 && value < 0.01
      ? "<0.01"
      : value.toLocaleString("en-US", { maximumFractionDigits: 2 });
  return { value: formatted, unit };
}

export function UsageChart({ tree }: { tree: ComputeTree }) {
  const [metric, setMetric] = useState<UsageMetric>("egressGiB");
  const [groupBy, setGroupBy] = useState<DeployUsageTimeseriesGroup>("app");
  const [interval, setInterval] = useState<DeployUsageTimeseriesInterval>("day");
  const [projectId, setProjectId] = useState(ALL_PROJECTS);
  const [scopeFilters, setScopeFilters] = useState<UsageScopeFilter[]>([]);
  const [isScopeOpen, setIsScopeOpen] = useState(false);
  const [monthsAgo, setMonthsAgo] = useState<MonthsAgo>(0);
  const [requestedDayStart, setRequestedDayStart] = useState<number>();
  const [showDeployments, setShowDeployments] = useState(true);
  const now = useMemo(() => new Date(), []);
  const period = billingPeriod(now, monthsAgo);
  const availableDays = useMemo(
    () => daysInPeriod(period.start, period.end),
    [period.start, period.end],
  );
  const selectedDayStart =
    requestedDayStart !== undefined && availableDays.includes(requestedDayStart)
      ? requestedDayStart
      : (availableDays.at(-1) ?? period.start);
  const selectedDayEnd =
    selectedDayStart === currentBucketStart(now, "day") ? now.getTime() : selectedDayStart + DAY_MS;
  const chartPeriod =
    interval === "hour"
      ? { start: selectedDayStart, end: selectedDayEnd }
      : { start: period.start, end: period.end };
  const periodOptions = PERIODS.map((value) => ({
    value: value.toString(),
    label: billingPeriod(now, value).label,
  }));
  const scopeProjects = useMemo(
    () =>
      projectId === ALL_PROJECTS
        ? tree.projects
        : tree.projects.filter((project) => project.projectId === projectId),
    [projectId, tree.projects],
  );
  const scopeApps = useMemo(
    () =>
      scopeProjects.flatMap((project) =>
        project.apps.map((app) => ({
          appId: app.appId,
          name: projectId === ALL_PROJECTS ? `${project.name} / ${app.name}` : app.name,
        })),
      ),
    [projectId, scopeProjects],
  );
  const scopeEnvironments = useMemo(
    () =>
      scopeProjects.flatMap((project) =>
        project.apps.flatMap((app) =>
          app.environments.map((environment) => ({
            id: environment.environmentId,
            appId: app.appId,
            slug: environment.name,
          })),
        ),
      ),
    [scopeProjects],
  );
  const selection = useMemo(() => getAppEnvironmentSelection(scopeFilters), [scopeFilters]);
  const scope = {
    projectId: projectId === ALL_PROJECTS ? "" : projectId,
    appIds: [...selection.appIds],
    environmentIds: [...selection.environmentIds],
  };

  const timeseries = trpc.billing.queryDeployUsageTimeseries.useQuery(
    interval === "hour"
      ? {
          interval,
          day: selectedDayStart,
          groupBy,
          scope,
          monthsAgo,
        }
      : {
          interval,
          groupBy,
          scope,
          monthsAgo,
        },
    {
      trpc: { context: { skipBatch: true } },
      retry: 1,
    },
  );
  const dailyOverview = trpc.billing.queryDeployUsageTimeseries.useQuery(
    {
      interval: "day",
      groupBy: "total",
      scope,
      monthsAgo,
    },
    {
      enabled: interval === "hour",
      trpc: { context: { skipBatch: true } },
      retry: 1,
    },
  );
  const deploymentAnnotations = trpc.billing.queryDeployUsageAnnotations.useQuery(
    interval === "hour"
      ? { interval, day: selectedDayStart, scope, monthsAgo }
      : { interval, scope, monthsAgo },
    {
      enabled: showDeployments,
      trpc: { context: { skipBatch: true } },
      retry: 1,
    },
  );
  const latestReportedHour = timeseries.data?.reduce(
    (latest, row) => Math.max(latest, row.time),
    Number.NEGATIVE_INFINITY,
  );
  const chartDataEnd =
    interval === "hour" &&
    selectedDayStart === currentBucketStart(now, "day") &&
    latestReportedHour !== undefined &&
    Number.isFinite(latestReportedHour)
      ? Math.min(chartPeriod.end, latestReportedHour + HOUR_MS)
      : chartPeriod.end;
  const chart = buildUsageChart({
    rows: timeseries.data ?? [],
    metric,
    groupBy,
    interval,
    tree,
    start: chartPeriod.start,
    end: chartDataEnd,
  });
  const chartConfig: ChartConfig = Object.fromEntries(
    chart.series.map((series) => [series.key, { label: series.label, color: series.color }]),
  );
  const overviewEnd = (availableDays.at(-1) ?? period.start) + DAY_MS;
  const overview = buildUsageChart({
    rows: dailyOverview.data ?? [],
    metric,
    groupBy: "total",
    interval: "day",
    tree,
    start: period.start,
    end: overviewEnd,
  });
  const overviewConfig: ChartConfig = Object.fromEntries(
    overview.series.map((series) => [series.key, { label: series.label, color: series.color }]),
  );
  const metricDetails = METRICS[metric];
  const incompleteFrom =
    monthsAgo !== 0
      ? undefined
      : interval === "day" || selectedDayStart === currentBucketStart(now, "day")
        ? currentBucketStart(now, interval)
        : undefined;
  const deploymentsByTime = new Map(
    (showDeployments ? (deploymentAnnotations.data ?? []) : []).map((annotation) => [
      annotation.time,
      annotation,
    ]),
  );
  const showDeploymentLabels = (deploymentAnnotations.data?.length ?? 0) <= MAX_DEPLOYMENT_LABELS;

  return (
    <div>
      <div className="flex items-center justify-between gap-4 px-4 py-3">
        <span className="text-xs text-gray-11">
          {interval === "day"
            ? `All instances in ${period.label}`
            : `All instances on ${formatDay(selectedDayStart)}`}
        </span>
        <div className="flex shrink-0 items-baseline gap-1 tabular-nums">
          {timeseries.isLoading ? (
            <Skeleton className="h-6 w-20" />
          ) : timeseries.isError ? (
            <span className="font-semibold text-2xl text-gray-12 leading-tight">—</span>
          ) : (
            <>
              <span className="font-semibold text-2xl text-gray-12 leading-tight tracking-tight">
                {formatCompactQuantity(chart.total)}
              </span>
              <span className="text-xs text-gray-10">{metricDetails.unit}</span>
            </>
          )}
        </div>
      </div>
      <ItemSeparator />
      <div className="flex flex-wrap items-end gap-3 px-4 py-3">
        <ChartSelect
          label="Period"
          value={monthsAgo.toString()}
          items={periodOptions}
          onValueChange={(value) => {
            const parsed = Number.parseInt(value, 10);
            if (isMonthsAgo(parsed)) {
              setMonthsAgo(parsed);
              setRequestedDayStart(undefined);
            }
          }}
        />
        <ChartSelect
          label="Metric"
          value={metric}
          items={Object.entries(METRICS).map(([value, item]) => ({ value, label: item.label }))}
          onValueChange={(value) => {
            if (isUsageMetric(value)) {
              setMetric(value);
            }
          }}
        />
        <ChartSelect
          label="Project"
          value={projectId}
          items={[
            { value: ALL_PROJECTS, label: "All projects" },
            ...tree.projects.map((project) => ({
              value: project.projectId,
              label: project.name,
            })),
          ]}
          className="min-w-36 max-w-64"
          onValueChange={(value) => {
            if (
              value === ALL_PROJECTS ||
              tree.projects.some((project) => project.projectId === value)
            ) {
              setProjectId(value);
              setScopeFilters([]);
            }
          }}
        />
        <ChartScope
          apps={scopeApps}
          environments={scopeEnvironments}
          filters={scopeFilters}
          updateFilters={setScopeFilters}
          open={isScopeOpen}
          onOpenChange={setIsScopeOpen}
        />
        <ChartSelect
          label="Group by"
          value={groupBy}
          items={Object.entries(GROUP_LABELS).map(([value, label]) => ({ value, label }))}
          onValueChange={(value) => {
            if (isUsageGroup(value)) {
              setGroupBy(value);
            }
          }}
        />
        <ChartSelect
          label="Interval"
          value={interval}
          items={Object.entries(INTERVAL_LABELS).map(([value, label]) => ({ value, label }))}
          onValueChange={(value) => {
            if (isUsageInterval(value)) {
              setInterval(value);
            }
          }}
        />
        <div className="flex min-w-32 flex-col gap-1">
          <span className="text-[11px] text-gray-10">Annotations</span>
          <div className="flex h-8 items-center gap-2 text-xs text-gray-11">
            <Switch
              checked={showDeployments}
              onCheckedChange={setShowDeployments}
              size="sm"
              aria-label="Show deployment annotations"
            />
            Deployments
          </div>
        </div>
      </div>
      <ItemSeparator />
      {interval === "hour" ? (
        <>
          <UsageDayNavigator
            days={availableDays}
            selectedDayStart={selectedDayStart}
            onSelectDay={setRequestedDayStart}
            periodLabel={period.label}
            data={overview.data}
            config={overviewConfig}
            isLoading={dailyOverview.isLoading}
            isError={dailyOverview.isError}
            incompleteFrom={monthsAgo === 0 ? currentBucketStart(now, "day") : undefined}
            start={period.start}
            end={overviewEnd}
          />
          <ItemSeparator />
        </>
      ) : null}
      <div className="px-4 pt-2 pb-4">
        <AreaTimeseriesChart
          data={chart.data}
          config={chartConfig}
          height={240}
          isLoading={timeseries.isLoading}
          isError={timeseries.isError}
          incompleteFrom={incompleteFrom}
          annotations={
            showDeployments
              ? (deploymentAnnotations.data ?? []).map((annotation) => ({
                  timestamp: annotation.time,
                  label: showDeploymentLabels
                    ? formatDeploymentLabel(annotation.deploymentIds[0], annotation.count)
                    : undefined,
                }))
              : undefined
          }
          renderTooltipFooter={(point) => {
            const annotation = deploymentsByTime.get(point.originalTimestamp);
            return annotation ? <DeploymentAnnotation annotation={annotation} /> : null;
          }}
          showDateInTooltip
          formatTooltipValue={(value) => formatUsageValue(value, metricDetails.unit)}
          axis={{
            x: { domain: [chartPeriod.start, chartPeriod.end], utc: true },
            y: {
              floor: 0,
              width: 76,
              formatTick: (value) =>
                value <= 0 ? "" : `${formatCompactQuantity(value)} ${metricDetails.axisUnit}`,
            },
          }}
        />
        {chart.series.length > 0 ? (
          <div className="mt-3 flex flex-wrap gap-x-4 gap-y-2 px-9">
            {chart.series.map((series) => (
              <div
                key={series.key}
                className="flex min-w-0 items-center gap-1.5 text-xs text-gray-10"
              >
                <span
                  className="size-2 shrink-0 rounded-[2px]"
                  style={{ backgroundColor: series.color }}
                />
                <span className="max-w-48 truncate">{series.label}</span>
              </div>
            ))}
          </div>
        ) : null}
        {chart.hasOther ? (
          <p className="mt-2 px-9 text-[11px] text-gray-9">
            The seven largest groups are shown. All remaining usage is grouped as Other.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function formatDeploymentLabel(
  deploymentId: string | undefined,
  count: number,
): string | undefined {
  if (!deploymentId) {
    return undefined;
  }
  const shortenedId = shortenId(deploymentId, {
    startChars: 4,
    endChars: 4,
    separator: "…",
  });
  return count > 1 ? `${shortenedId} +${count - 1}` : shortenedId;
}

function DeploymentAnnotation({
  annotation,
}: {
  annotation: { count: number; deploymentIds: string[] };
}) {
  const remaining = annotation.count - annotation.deploymentIds.length;
  return (
    <div className="mt-0.5 grid gap-1 border-grayA-4 border-t pt-1.5">
      <div className="flex items-center justify-between gap-4 text-[11px]">
        <span className="font-medium text-feature-11">Deployments</span>
        <span className="font-mono tabular-nums text-gray-10">
          {annotation.count.toLocaleString("en-US")}
        </span>
      </div>
      {annotation.deploymentIds.map((deploymentId) => (
        <span
          key={deploymentId}
          className="font-mono text-[11px] text-gray-11"
          title={deploymentId}
        >
          {shortenId(deploymentId, { startChars: 8, endChars: 4 })}
        </span>
      ))}
      {remaining > 0 ? (
        <span className="text-[11px] text-gray-9">+{remaining.toLocaleString("en-US")} more</span>
      ) : null}
    </div>
  );
}

function UsageDayNavigator({
  days,
  selectedDayStart,
  onSelectDay,
  periodLabel,
  data,
  config,
  isLoading,
  isError,
  incompleteFrom,
  start,
  end,
}: {
  days: number[];
  selectedDayStart: number;
  onSelectDay: (day: number) => void;
  periodLabel: string;
  data: ReturnType<typeof buildUsageChart>["data"];
  config: ChartConfig;
  isLoading: boolean;
  isError: boolean;
  incompleteFrom?: number;
  start: number;
  end: number;
}) {
  return (
    <div className="px-4 py-3">
      <div className="mb-2 flex items-center justify-between gap-4 text-xs">
        <span className="font-medium text-gray-11">Daily overview · {periodLabel}</span>
        <span className="tabular-nums text-gray-10">{formatDay(selectedDayStart)}</span>
      </div>
      <div className="relative overflow-hidden rounded-md border border-grayA-4 bg-grayA-1">
        <AreaTimeseriesChart
          data={data}
          config={config}
          height={64}
          isLoading={isLoading}
          isError={isError}
          incompleteFrom={incompleteFrom}
          axis={{ visible: false, x: { domain: [start, end], utc: true }, y: { floor: 0 } }}
          paleFill
          hideTooltip
          showZeroLine
        />
        <div
          className="absolute inset-0 grid"
          style={{ gridTemplateColumns: `repeat(${days.length}, minmax(0, 1fr))` }}
        >
          {days.map((day) => {
            const selected = day === selectedDayStart;
            return (
              <button
                key={day}
                type="button"
                aria-label={`Show hourly usage for ${formatDay(day)}`}
                aria-pressed={selected}
                title={formatDay(day)}
                onClick={() => onSelectDay(day)}
                className={`min-w-0 border-grayA-3 border-l outline-none first:border-l-0 hover:bg-grayA-2 focus-visible:z-10 focus-visible:ring-1 focus-visible:ring-blueA-8 ${
                  selected ? "bg-blueA-3 ring-1 ring-inset ring-blueA-7" : ""
                }`}
              />
            );
          })}
        </div>
      </div>
      <div className="mt-1 flex justify-between font-mono text-[10px] text-gray-9">
        <span>{formatDay(days[0] ?? start, false)}</span>
        <span>{formatDay(days.at(-1) ?? end, false)}</span>
      </div>
    </div>
  );
}

function ChartScope({
  apps,
  environments,
  filters,
  updateFilters,
  open,
  onOpenChange,
}: {
  apps: Array<{ appId: string; name: string }>;
  environments: Array<{ id: string; appId: string; slug: string }>;
  filters: UsageScopeFilter[];
  updateFilters: (filters: UsageScopeFilter[]) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const triggerId = useId();
  const selection = getAppEnvironmentSelection(filters);
  const selectedCount = selection.appIds.size + selection.environmentIds.size;

  return (
    <div className="flex min-w-40 flex-col gap-1">
      <label htmlFor={triggerId} className="text-[11px] text-gray-10">
        Scope
      </label>
      <Popover open={open} onOpenChange={onOpenChange}>
        <PopoverTrigger
          render={
            <button
              id={triggerId}
              type="button"
              className="relative flex h-8 w-full items-center rounded-lg border border-grayA-4 bg-transparent px-3 text-left text-xs shadow-sm outline-none hover:bg-grayA-2 focus-visible:ring-1 focus-visible:ring-grayA-8"
            >
              <span className="truncate">
                {selectedCount === 0 ? "All apps" : `${selectedCount} selected`}
              </span>
              <ChevronExpandY className="absolute right-2.5 size-3 text-gray-9" />
            </button>
          }
        />
        <PopoverContent align="start" className="w-auto p-0">
          <AppEnvironmentFilterList
            apps={apps}
            environments={environments}
            filters={filters}
            updateFilters={updateFilters}
            createFilter={createUsageScopeFilter}
            onApply={() => onOpenChange(false)}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

function ChartSelect({
  label,
  value,
  items,
  onValueChange,
  className = "min-w-32",
  contentClassName,
}: {
  label: string;
  value: string;
  items: Array<{ value: string; label: string }>;
  onValueChange: (value: string) => void;
  className?: string;
  contentClassName?: string;
}) {
  const triggerId = useId();

  return (
    <div className={`flex flex-col gap-1 ${className}`}>
      <label htmlFor={triggerId} className="text-[11px] text-gray-10">
        {label}
      </label>
      <Select
        value={value}
        items={items}
        onValueChange={(newValue) => {
          if (newValue !== null) {
            onValueChange(newValue);
          }
        }}
      >
        <SelectTrigger
          id={triggerId}
          className="h-8 min-h-0! rounded-lg border-grayA-4 bg-transparent text-xs shadow-sm focus:ring-0"
          rightIcon={<ChevronExpandY className="absolute right-2.5 size-3 text-gray-9" />}
        >
          <SelectValue className="truncate text-xs" />
        </SelectTrigger>
        <SelectContent className={contentClassName}>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value} className="text-xs">
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
