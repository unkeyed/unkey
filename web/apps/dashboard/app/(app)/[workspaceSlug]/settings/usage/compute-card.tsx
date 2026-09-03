"use client";

import { AreaTimeseriesChart, type ValueParts } from "@/components/charts/area-timeseries";
import { formatCompactQuantity, formatPrice } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import type { DeployUsageTimeseries } from "@unkey/clickhouse";
import { ChartActivity, ChevronRight, Cube } from "@unkey/icons";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  Skeleton,
} from "@unkey/ui";
import { Fragment, type ReactNode, useMemo, useState } from "react";
import {
  type ComputeTree,
  type UsageApp,
  type UsageCostsCents,
  type UsageProject,
  type UsageQuantities,
  microCentsToDisplayCents,
  priceUsageQuantitiesCents,
} from "./compute-tree";
import { UsageChart } from "./usage-chart";
import { type UsageMetric, buildUsageMetricChartData } from "./usage-chart-data";

const RESOURCE_COLUMNS: ReadonlyArray<{
  key: keyof UsageQuantities;
  costKey: keyof UsageCostsCents;
  label: string;
  unit: string;
  width: string;
}> = [
  { key: "cpuHours", costKey: "cpu", label: "CPU", unit: "hrs", width: "w-16" },
  {
    key: "memoryGiBHours",
    costKey: "memory",
    label: "Memory",
    unit: "GiB-hrs",
    width: "w-24",
  },
  { key: "egressGiB", costKey: "egress", label: "Egress", unit: "GiB", width: "w-20" },
  {
    key: "diskGiBHours",
    costKey: "disk",
    label: "Storage",
    unit: "GiB-hrs",
    width: "w-24",
  },
];

const SKELETON_ROWS = ["first", "second", "third"];
const PROJECT_METRICS: ReadonlyArray<{
  key: UsageMetric;
  costKey: keyof UsageCostsCents;
  label: string;
  unit: string;
  color: string;
}> = [
  {
    key: "cpuHours",
    costKey: "cpu",
    label: "CPU",
    unit: "hours",
    color: "hsl(var(--feature-8))",
  },
  {
    key: "memoryGiBHours",
    costKey: "memory",
    label: "Memory",
    unit: "GiB-hours",
    color: "hsl(var(--info-8))",
  },
  {
    key: "egressGiB",
    costKey: "egress",
    label: "Public egress",
    unit: "GiB",
    color: "hsl(var(--error-8))",
  },
  {
    key: "diskGiBHours",
    costKey: "disk",
    label: "Storage",
    unit: "GiB-hours",
    color: "hsl(var(--warning-8))",
  },
];
const COSTS_PER_USAGE_UNIT = priceUsageQuantitiesCents({
  cpuHours: 1,
  memoryGiBHours: 1,
  egressGiB: 1,
  diskGiBHours: 1,
});

export function ComputeCardShell({
  description,
  amount,
  children,
}: {
  description: string;
  amount?: ReactNode;
  children: ReactNode;
}) {
  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-orangeA-3 text-orange-11">
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Compute</ItemTitle>
          <ItemDescription>{description}</ItemDescription>
        </ItemContent>
        {amount === undefined ? null : (
          <ItemActions className="font-semibold text-2xl text-gray-12 leading-tight tracking-tight tabular-nums">
            {amount}
          </ItemActions>
        )}
      </ItemHeader>
      <ItemSeparator />
      {children}
    </ItemGroup>
  );
}

export function ComputeCardSkeleton() {
  return (
    <ComputeCardShell
      description="Usage per project this period"
      amount={<Skeleton className="h-6 w-20" />}
    >
      {SKELETON_ROWS.map((row, index) => (
        <Fragment key={row}>
          {index === 0 ? null : <ItemSeparator />}
          <Item className="gap-2">
            <ChevronRight iconSize="sm-regular" className="shrink-0 text-gray-6" />
            <ItemMedia className="size-5 border border-grayA-4 bg-gray-1">
              <Cube />
            </ItemMedia>
            <ItemContent>
              <Skeleton className="h-3 w-40" />
            </ItemContent>
            <ItemActions className="w-20 justify-end">
              <Skeleton className="h-3 w-12" />
            </ItemActions>
          </Item>
        </Fragment>
      ))}
    </ComputeCardShell>
  );
}

type ComputeCardProps = {
  tree: ComputeTree;
};

export function ComputeCard({ tree }: ComputeCardProps) {
  const [usageOpen, setUsageOpen] = useState(false);
  const [open, setOpen] = useState<ReadonlySet<string>>(new Set());
  const now = useMemo(() => new Date(), []);
  const periodStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
  const currentDayStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  const timeseries = trpc.billing.queryDeployUsageTimeseries.useQuery(
    {
      interval: "day",
      groupBy: "project",
      scope: { projectId: "", appIds: [], environmentIds: [] },
      monthsAgo: 0,
    },
    {
      enabled: open.size > 0,
      trpc: { context: { skipBatch: true } },
      retry: 1,
      staleTime: 30_000,
    },
  );
  const usageByProject = useMemo(
    () => Map.groupBy(timeseries.data ?? [], (row) => row.groupId),
    [timeseries.data],
  );
  const hasComputeUsage = tree.projects.some((project) => project.apps.length > 0);

  const toggle = (projectId: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (!next.delete(projectId)) {
        next.add(projectId);
      }
      return next;
    });

  return (
    <ComputeCardShell
      description="Usage per project this period"
      amount={formatPrice(microCentsToDisplayCents(tree.microCents))}
    >
      {tree.projects.length === 0 ? (
        <Item>
          <ItemContent>
            <ItemDescription>No compute usage recorded this period.</ItemDescription>
          </ItemContent>
        </Item>
      ) : (
        <>
          {hasComputeUsage ? (
            <>
              <Item
                className="gap-2"
                render={
                  <button
                    type="button"
                    aria-expanded={usageOpen}
                    onClick={() => setUsageOpen((current) => !current)}
                  />
                }
              >
                <ChevronRight
                  iconSize="sm-regular"
                  className={`shrink-0 text-gray-9 transition-transform duration-150 ease-out motion-reduce:transition-none ${usageOpen ? "rotate-90" : ""}`}
                />
                <ItemMedia className="size-5 bg-blueA-3 text-blue-11">
                  <ChartActivity />
                </ItemMedia>
                <ItemContent>
                  <ItemTitle>Usage over time</ItemTitle>
                  <ItemDescription>Filter and group instance usage</ItemDescription>
                </ItemContent>
              </Item>
              {usageOpen ? (
                <>
                  <ItemSeparator />
                  <UsageChart tree={tree} />
                </>
              ) : null}
              <ItemSeparator className="bg-gray-5" />
            </>
          ) : null}
          {tree.projects.map((project, index) => (
            <Fragment key={project.projectId}>
              {index === 0 ? null : <ItemSeparator className="bg-gray-5" />}
              <ProjectRow
                project={project}
                open={open.has(project.projectId)}
                onToggle={() => toggle(project.projectId)}
                usage={usageByProject.get(project.projectId) ?? []}
                usageStart={periodStart}
                usageEnd={now.getTime()}
                incompleteFrom={currentDayStart}
                isUsageLoading={timeseries.isLoading}
                isUsageError={timeseries.isError}
              />
            </Fragment>
          ))}
        </>
      )}
    </ComputeCardShell>
  );
}

function ProjectRow({
  project,
  open,
  onToggle,
  usage,
  usageStart,
  usageEnd,
  incompleteFrom,
  isUsageLoading,
  isUsageError,
}: {
  project: UsageProject;
  open: boolean;
  onToggle: () => void;
  usage: DeployUsageTimeseries[];
  usageStart: number;
  usageEnd: number;
  incompleteFrom: number;
  isUsageLoading: boolean;
  isUsageError: boolean;
}) {
  return (
    <div>
      <Item
        className="gap-2"
        render={<button type="button" aria-expanded={open} onClick={onToggle} />}
      >
        <ChevronRight
          iconSize="sm-regular"
          className={`shrink-0 text-gray-9 transition-transform duration-150 ease-out motion-reduce:transition-none ${open ? "rotate-90" : ""}`}
        />
        <ItemMedia className="size-5 border border-grayA-4 bg-gray-1">
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle className="truncate">{project.name}</ItemTitle>
        </ItemContent>
        <ItemActions className="w-20 justify-end font-medium tabular-nums">
          {formatPrice(microCentsToDisplayCents(project.microCents))}
        </ItemActions>
      </Item>
      <div
        className="grid transition-[grid-template-rows] duration-200 ease-out motion-reduce:transition-none"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          {project.apps.length === 0 ? null : (
            <>
              {open ? (
                <ProjectUsageCharts
                  project={project}
                  usage={usage}
                  start={usageStart}
                  end={usageEnd}
                  incompleteFrom={incompleteFrom}
                  isLoading={isUsageLoading}
                  isError={isUsageError}
                />
              ) : null}
              <Band>
                <div className="min-w-0 flex-1">App</div>
                {RESOURCE_COLUMNS.map((column) => (
                  <div key={column.key} className={`${column.width} text-right`}>
                    {column.label}
                  </div>
                ))}
                <div className="w-20 text-right">Total</div>
              </Band>
              {project.apps.map((app) => (
                <AppRows key={app.appId} app={app} />
              ))}
            </>
          )}
          <Band>
            <div className="min-w-0 flex-1">Gateway</div>
            <div className="w-24 text-right">Keys</div>
            <div className="w-20 text-right">Cost</div>
          </Band>
          <div className="flex items-center gap-3 px-4 py-2.5">
            <span className="min-w-0 flex-1 truncate font-medium text-[13px] text-gray-12">
              Verified keys
            </span>
            <span className="w-24 text-right text-[13px] text-gray-11 tabular-nums">
              {project.gateway.activeKeys.toLocaleString("en-US")}
            </span>
            <span className="w-20 text-right font-medium text-[13px] text-gray-12 tabular-nums">
              {formatPrice(microCentsToDisplayCents(project.gateway.microCents))}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function ProjectUsageCharts({
  project,
  usage,
  start,
  end,
  incompleteFrom,
  isLoading,
  isError,
}: {
  project: UsageProject;
  usage: DeployUsageTimeseries[];
  start: number;
  end: number;
  incompleteFrom: number;
  isLoading: boolean;
  isError: boolean;
}) {
  const costs = priceUsageQuantitiesCents(project);

  return (
    <div className="grid gap-px border-grayA-4 border-y bg-grayA-4 sm:grid-cols-2 xl:grid-cols-4">
      {PROJECT_METRICS.map((metric) => (
        <div key={metric.key} className="min-w-0 bg-gray-1 px-3 pt-3 pb-2">
          <div className="mb-1 flex items-start justify-between gap-2">
            <span className="font-medium text-gray-11 text-xs">{metric.label}</span>
            <span className="min-w-0 text-right tabular-nums">
              <span className="block truncate font-medium text-gray-12 text-xs">
                {formatPrice(costs[metric.costKey])}
              </span>
              <span className="block truncate text-[10px] text-gray-9 leading-3">
                {formatCompactQuantity(project[metric.key])} {metric.unit}
              </span>
            </span>
          </div>
          <AreaTimeseriesChart
            data={buildUsageMetricChartData({
              rows: usage,
              metric: metric.key,
              interval: "day",
              start,
              end,
            }).map((point) => ({
              ...point,
              value: Number(point.value) * COSTS_PER_USAGE_UNIT[metric.costKey],
            }))}
            config={{ value: { label: metric.label, color: metric.color } }}
            height={64}
            isLoading={isLoading}
            isError={isError}
            incompleteFrom={incompleteFrom}
            axis={null}
            paleFill
            showDateInTooltip
            showZeroLine
            formatTooltipValue={(value) =>
              formatCostValue(value, metric.unit, COSTS_PER_USAGE_UNIT[metric.costKey])
            }
          />
        </div>
      ))}
    </div>
  );
}

function formatCostValue(costCents: number, unit: string, costPerUnitCents: number): ValueParts {
  const usage = costCents / costPerUnitCents;
  return {
    value: formatPrice(costCents),
    hint: `(${formatCompactQuantity(usage)} ${unit})`,
  };
}

function AppRows({ app }: { app: UsageApp }) {
  return (
    <div>
      <div className="flex items-center gap-3 px-4 pt-2.5 pb-1">
        <span className="min-w-0 flex-1 truncate font-medium text-[13px] text-gray-12">
          {app.name}
        </span>
        <ResourceCosts usage={app} className="text-[13px] text-gray-11" />
        <span className="w-20 text-right font-medium text-[13px] text-gray-12 tabular-nums">
          {formatPrice(microCentsToDisplayCents(app.microCents))}
        </span>
      </div>
      {app.environments.map((environment) => (
        <div
          key={environment.environmentId}
          className="flex items-center gap-3 px-4 py-1 last:pb-2.5"
        >
          <span className="min-w-0 flex-1 truncate text-gray-10 text-xs">{environment.name}</span>
          <ResourceCosts usage={environment} className="text-gray-10 text-xs" />
          <span className="w-20 text-right text-gray-11 text-xs tabular-nums">
            {formatPrice(microCentsToDisplayCents(environment.microCents))}
          </span>
        </div>
      ))}
    </div>
  );
}

function Band({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-center gap-3 border-grayA-4 border-y bg-grayA-2 px-4 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider">
      {children}
    </div>
  );
}

function ResourceCosts({ usage, className }: { usage: UsageQuantities; className: string }) {
  const costs = priceUsageQuantitiesCents(usage);

  return (
    <>
      {RESOURCE_COLUMNS.map((column) => (
        <span
          key={column.key}
          className={`${column.width} text-right tabular-nums ${className}`}
          aria-label={`${column.label}: ${formatPrice(costs[column.costKey])}, ${formatCompactQuantity(usage[column.key])} ${column.unit}`}
        >
          <span className="block">{formatPrice(costs[column.costKey])}</span>
          <span className="block text-[10px] text-gray-9 leading-3">
            {formatCompactQuantity(usage[column.key])} {column.unit}
          </span>
        </span>
      ))}
    </>
  );
}
