"use client";

import { formatCompactQuantity, formatPrice } from "@/lib/fmt";
import { ChartActivity, ChevronRight, Cube } from "@unkey/icons";
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
import dynamic from "next/dynamic";
import { Fragment, type ReactNode, useState } from "react";
import {
  type ComputeTree,
  type UsageApp,
  type UsageCostsCents,
  type UsageProject,
  type UsageQuantities,
  microCentsToDisplayCents,
  priceUsageQuantitiesCents,
} from "./compute-tree";

const RESOURCE_COLUMNS: ReadonlyArray<{
  key: keyof UsageQuantities;
  costKey: keyof UsageCostsCents;
  label: string;
  spendLabel: string;
  unit: string;
  width: string;
  color: string;
}> = [
  {
    key: "cpuHours",
    costKey: "cpu",
    label: "CPU",
    spendLabel: "CPU",
    unit: "vCPU-hours",
    width: "w-16",
    color: "hsl(var(--feature-8))",
  },
  {
    key: "memoryGiBHours",
    costKey: "memory",
    label: "Memory",
    spendLabel: "Memory",
    unit: "GiB-hours",
    width: "w-24",
    color: "hsl(var(--info-8))",
  },
  {
    key: "egressGiB",
    costKey: "egress",
    label: "Egress",
    spendLabel: "Public egress",
    unit: "GiB",
    width: "w-20",
    color: "hsl(var(--error-8))",
  },
  {
    key: "diskGiBHours",
    costKey: "disk",
    label: "Storage",
    spendLabel: "Storage",
    unit: "GiB-hours",
    width: "w-24",
    color: "hsl(var(--warning-8))",
  },
];

const SKELETON_ROWS = ["first", "second", "third"];
const LazyUsageChart = dynamic(() => import("./usage-chart").then(({ UsageChart }) => UsageChart), {
  loading: UsageChartSkeleton,
  ssr: false,
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
                  <LazyUsageChart tree={tree} />
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
}: {
  project: UsageProject;
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
              {open ? <ProjectSpendByResource project={project} /> : null}
              <Band>
                <div className="min-w-0 flex-1">App</div>
                {RESOURCE_COLUMNS.map((column) => (
                  <div key={column.key} className={`${column.width} text-right`}>
                    {column.label}
                  </div>
                ))}
                <div className="w-20 text-right">Total</div>
              </Band>
              <div className="max-h-96 overflow-y-auto overscroll-contain [scrollbar-width:thin]">
                {project.apps.map((app) => (
                  <AppRows key={app.appId} app={app} />
                ))}
              </div>
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

function ProjectSpendByResource({ project }: { project: UsageProject }) {
  const costs = priceUsageQuantitiesCents(project);
  const resources = RESOURCE_COLUMNS.map((resource) => ({
    ...resource,
    cost: costs[resource.costKey],
  }));
  const totalCost = resources.reduce((sum, resource) => sum + resource.cost, 0);

  return (
    <div className="border-grayA-4 border-y px-4 py-3">
      <div className="mb-2 font-medium text-gray-11 text-xs">Spend by resource</div>
      <div
        className="flex h-1.5 overflow-hidden rounded-full bg-grayA-3"
        role="img"
        aria-label={`Spend by resource: ${resources
          .map((resource) => `${resource.spendLabel} ${formatPrice(resource.cost)}`)
          .join(", ")}`}
      >
        {totalCost > 0
          ? resources.map((resource) =>
              resource.cost > 0 ? (
                <span
                  key={resource.key}
                  className="min-w-px basis-0"
                  style={{ backgroundColor: resource.color, flexGrow: resource.cost }}
                />
              ) : null,
            )
          : null}
      </div>
      <div className="mt-2.5 flex flex-wrap gap-x-5 gap-y-1.5">
        {resources.map((resource) => (
          <div key={resource.key} className="flex items-center gap-1.5 text-xs">
            <span
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: resource.color }}
            />
            <span className="text-gray-10">{resource.spendLabel}</span>
            <span className="font-medium text-gray-12 tabular-nums">
              {formatPrice(resource.cost)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function UsageChartSkeleton() {
  return (
    <div aria-label="Loading usage chart">
      <div className="flex items-center justify-between gap-4 px-4 py-3">
        <Skeleton className="h-3 w-40" />
        <Skeleton className="h-6 w-20" />
      </div>
      <ItemSeparator />
      <div className="flex flex-wrap gap-3 px-4 py-3">
        {SKELETON_ROWS.map((row) => (
          <div key={row} className="grid w-32 gap-1">
            <Skeleton className="h-2.5 w-12" />
            <Skeleton className="h-8 w-full rounded-lg" />
          </div>
        ))}
      </div>
      <ItemSeparator />
      <div className="px-4 pt-2 pb-4">
        <Skeleton className="h-60 w-full rounded-md" />
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
        <InfoTooltip
          key={column.key}
          content={
            <span className="whitespace-nowrap tabular-nums">
              {formatCompactQuantity(usage[column.key])} {column.unit}
            </span>
          }
          position={{ side: "top" }}
          delayDuration={150}
          asChild
        >
          <span
            className={`${column.width} text-right tabular-nums ${className}`}
            aria-label={`${column.label}: ${formatPrice(costs[column.costKey])}, ${formatCompactQuantity(usage[column.key])} ${column.unit}`}
          >
            {formatPrice(costs[column.costKey])}
          </span>
        </InfoTooltip>
      ))}
    </>
  );
}
