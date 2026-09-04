"use client";

import { SPEND_BAR_CHART_HEIGHT, SpendBarChart } from "@/components/charts/spend-bar-chart";
import { DEPLOY_METER_RATES } from "@/lib/billing/deployPricing";
import { formatCompactQuantity, formatPrice } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { ChevronRight, Cube } from "@unkey/icons";
import {
  InfoTooltip,
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
  type UsageGateway,
  type UsageProject,
  type UsageQuantities,
  microCentsToDisplayCents,
  priceUsageQuantitiesCents,
} from "./compute-tree";
import { buildSpendSeries } from "./spend-series";

const METERS: ReadonlyArray<{
  key: keyof UsageQuantities;
  costKey: keyof UsageCostsCents;
  label: string;
  columnLabel?: string;
  unit: string;
  barClass: string;
  width: string;
}> = [
  {
    key: "cpuHours",
    costKey: "cpu",
    label: "CPU",
    unit: "vCPU-hrs",
    barClass: "bg-feature-8",
    width: "w-16",
  },
  {
    key: "memoryGiBHours",
    costKey: "memory",
    label: "Memory",
    unit: "GiB-hrs",
    barClass: "bg-info-8",
    width: "w-24",
  },
  {
    key: "egressGiB",
    costKey: "egress",
    label: "Public egress",
    columnLabel: "Egress",
    unit: "GiB",
    barClass: "bg-error-8",
    width: "w-20",
  },
  {
    key: "diskGiBHours",
    costKey: "disk",
    label: "Storage",
    unit: "GiB-hrs",
    barClass: "bg-warning-8",
    width: "w-24",
  },
];

const SKELETON_ROWS = ["first", "second", "third"];
const ALL_PROJECTS = { projectId: "", appIds: [], environmentIds: [] };

export function ComputeCardShell({
  description,
  amount,
  chart,
  children,
}: {
  description: string;
  amount?: ReactNode;
  chart?: ReactNode;
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
      {chart === undefined ? (
        <ItemSeparator />
      ) : (
        <>
          <div className="px-4 pb-2">{chart}</div>
          <ItemSeparator className="bg-gray-5" />
        </>
      )}
      {children}
    </ItemGroup>
  );
}

export function ComputeCardSkeleton() {
  return (
    <ComputeCardShell
      description="Usage per project this period"
      amount={<Skeleton className="h-6 w-20" />}
      chart={<Skeleton className="w-full rounded-md" style={{ height: SPEND_BAR_CHART_HEIGHT }} />}
    >
      {SKELETON_ROWS.map((row, index) => (
        <Fragment key={row}>
          {index === 0 ? null : <ItemSeparator />}
          <Item className="gap-2">
            <ChevronRight iconSize="sm-regular" className="shrink-0 text-gray-6" />
            <Skeleton className="size-2 shrink-0 rounded-full" />
            <ItemContent>
              <Skeleton className="h-4 w-40" />
            </ItemContent>
            <ItemActions className="w-20 justify-end">
              <Skeleton className="h-4 w-12" />
            </ItemActions>
          </Item>
        </Fragment>
      ))}
    </ComputeCardShell>
  );
}

export function ComputeCard({ tree }: { tree: ComputeTree }) {
  const [open, setOpen] = useState<ReadonlySet<string>>(new Set());
  const now = useMemo(() => new Date(), []);
  const periodStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1);
  const currentDayStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  const hasComputeUsage = tree.projects.some((project) => project.apps.length > 0);
  const timeseries = trpc.billing.queryDeployUsageTimeseries.useQuery(
    { interval: "day", groupBy: "project", scope: ALL_PROJECTS, monthsAgo: 0 },
    {
      enabled: hasComputeUsage,
      trpc: { context: { skipBatch: true } },
      retry: 1,
      staleTime: 30_000,
    },
  );
  const spend = useMemo(
    () =>
      buildSpendSeries({
        tree,
        rows: timeseries.data ?? [],
        start: periodStart,
        end: now.getTime(),
      }),
    [tree, timeseries.data, periodStart, now],
  );

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
      chart={
        hasComputeUsage ? (
          <SpendBarChart
            data={spend.points}
            series={spend.series}
            incompleteFrom={currentDayStart}
            isLoading={timeseries.isLoading}
            isError={timeseries.isError}
          />
        ) : undefined
      }
    >
      {tree.projects.length === 0 ? (
        <Item>
          <ItemContent>
            <ItemDescription>No compute usage recorded this period.</ItemDescription>
          </ItemContent>
        </Item>
      ) : (
        tree.projects.map((project, index) => (
          <Fragment key={project.projectId}>
            {index === 0 ? null : <ItemSeparator className="bg-gray-5" />}
            <ProjectRow
              project={project}
              color={spend.series[index].color}
              open={open.has(project.projectId)}
              onToggle={() => toggle(project.projectId)}
            />
          </Fragment>
        ))
      )}
    </ComputeCardShell>
  );
}

function ProjectRow({
  project,
  color,
  open,
  onToggle,
}: {
  project: UsageProject;
  color: string;
  open: boolean;
  onToggle: () => void;
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
        <span
          className="size-2 shrink-0 rounded-full"
          style={{ backgroundColor: color }}
          aria-hidden="true"
        />
        <ItemContent>
          <ItemTitle className="truncate">{project.name}</ItemTitle>
        </ItemContent>
        <ItemActions className="w-20 justify-end font-medium tabular-nums">
          <TotalCost
            cents={microCentsToDisplayCents(project.microCents)}
            usage={project}
            gateway={project.gateway}
          />
        </ItemActions>
      </Item>
      <div
        className="grid transition-[grid-template-rows] duration-200 ease-out motion-reduce:transition-none"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          {project.apps.length === 0 ? null : (
            <>
              <ResourceBar usage={project} />
              <Band>
                <div className="min-w-0 flex-1">App</div>
                {METERS.map((meter) => (
                  <div key={meter.key} className={`${meter.width} text-right`}>
                    {meter.columnLabel ?? meter.label}
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

function ResourceBar({ usage }: { usage: UsageQuantities }) {
  const costs = priceUsageQuantitiesCents(usage);

  return (
    <div className="px-4 pt-3 pb-3.5">
      <div className="pb-2 text-[13px] text-gray-12">Spend by resource</div>
      <div className="flex h-1.5 w-full overflow-hidden rounded-full bg-grayA-3">
        {METERS.map((meter) => (
          <div
            key={meter.key}
            className={`basis-0 ${meter.barClass}`}
            style={{ flexGrow: costs[meter.costKey], minWidth: costs[meter.costKey] > 0 ? 3 : 0 }}
          />
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 pt-2.5">
        {METERS.map((meter) => (
          <InfoTooltip
            key={meter.key}
            asChild
            delayDuration={120}
            variant="inverted"
            position={{ side: "top" }}
            content={
              <span className="whitespace-nowrap tabular-nums">
                {formatCompactQuantity(usage[meter.key])} {meter.unit} ·{" "}
                {DEPLOY_METER_RATES[meter.costKey]}
              </span>
            }
          >
            <span className="flex items-center gap-2 text-[13px]">
              <span
                className={`size-2 shrink-0 rounded-full ${meter.barClass}`}
                aria-hidden="true"
              />
              <span className="text-gray-11">{meter.label}</span>
              <span className="text-gray-12 tabular-nums">{formatPrice(costs[meter.costKey])}</span>
              <span className="sr-only">
                {formatCompactQuantity(usage[meter.key])} {meter.unit} at{" "}
                {DEPLOY_METER_RATES[meter.costKey]}
              </span>
            </span>
          </InfoTooltip>
        ))}
      </div>
    </div>
  );
}

function AppRows({ app }: { app: UsageApp }) {
  return (
    <div>
      <div className="flex items-center gap-3 px-4 pt-2.5 pb-1">
        <span className="min-w-0 flex-1 truncate font-medium text-[13px] text-gray-12">
          {app.name}
        </span>
        <MeterCosts usage={app} className="text-[13px] text-gray-12" />
        <TotalCost
          cents={microCentsToDisplayCents(app.microCents)}
          usage={app}
          className="w-20 text-right font-medium text-[13px] text-gray-12"
        />
      </div>
      {app.environments.map((environment) => (
        <div
          key={environment.environmentId}
          className="flex items-center gap-3 px-4 py-1 last:pb-2.5"
        >
          <span className="min-w-0 flex-1 truncate text-gray-10 text-xs">{environment.name}</span>
          <MeterCosts usage={environment} className="text-gray-10 text-xs" />
          <TotalCost
            cents={microCentsToDisplayCents(environment.microCents)}
            usage={environment}
            className="w-20 text-right text-gray-11 text-xs"
          />
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

function MeterCosts({ usage, className }: { usage: UsageQuantities; className: string }) {
  const costs = priceUsageQuantitiesCents(usage);

  return (
    <>
      {METERS.map((meter) => {
        const amount = `${formatCompactQuantity(usage[meter.key])} ${meter.unit}`;
        return (
          <InfoTooltip
            key={meter.key}
            asChild
            delayDuration={120}
            variant="inverted"
            position={{ side: "top" }}
            content={<span className="whitespace-nowrap tabular-nums">{amount}</span>}
          >
            <span className={`${meter.width} shrink-0 text-right tabular-nums ${className}`}>
              {formatPrice(costs[meter.costKey])}
              <span className="sr-only">
                {meter.label}, {amount}
              </span>
            </span>
          </InfoTooltip>
        );
      })}
    </>
  );
}

function TotalCost({
  cents,
  usage,
  gateway,
  className,
}: {
  cents: number;
  usage: UsageQuantities;
  gateway?: UsageGateway;
  className?: string;
}) {
  const costs = priceUsageQuantitiesCents(usage);
  const rows: Array<{ key: string; label: string; cents: number; amount: string }> = METERS.map(
    (meter) => ({
      key: meter.key,
      label: meter.label,
      cents: costs[meter.costKey],
      amount: `${formatCompactQuantity(usage[meter.key])} ${meter.unit}`,
    }),
  );
  if (gateway !== undefined) {
    rows.push({
      key: "gateway",
      label: "Verified keys",
      cents: microCentsToDisplayCents(gateway.microCents),
      amount: `${gateway.activeKeys.toLocaleString("en-US")} keys`,
    });
  }

  return (
    <InfoTooltip
      asChild
      delayDuration={120}
      variant="inverted"
      position={{ side: "top" }}
      content={
        <div className="flex flex-col gap-1">
          {rows.map((row) => (
            <div key={row.key} className="flex items-baseline justify-between gap-4">
              <span className="opacity-75">{row.label}</span>
              <span className="tabular-nums">
                {formatPrice(row.cents)}
                <span className="pl-1.5 opacity-75">{row.amount}</span>
              </span>
            </div>
          ))}
        </div>
      }
    >
      <span className={`tabular-nums ${className ?? ""}`}>{formatPrice(cents)}</span>
    </InfoTooltip>
  );
}
