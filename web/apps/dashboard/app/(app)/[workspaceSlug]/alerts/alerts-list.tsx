"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, Cube, Pulse } from "@unkey/icons";
import {
  Button,
  Empty,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListItem,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Tabs,
  TabsList,
  TabsTrigger,
  TimestampInfo,
} from "@unkey/ui";
import Link from "next/link";
import { useMemo, useState } from "react";
import { AlertRowChart } from "./alert-row-chart";
import {
  alertMetricLabel,
  alertMetricOptions,
  formatAlertDistance,
  formatAlertValue,
  formatSigma,
  hasFixedAlertThreshold,
  isAlertMetric,
} from "./format";
import { ResolveAlertButton } from "./resolve-alert-button";
import { AlertStatusBadge } from "./status-badge";
import type { AlertMetric, AlertStatus } from "./types";

type StatusFilter = AlertStatus | "all";
type MetricFilter = AlertMetric | "all";

const STATUS_FILTERS: StatusFilter[] = ["open", "resolved", "all"];
const SKELETON_ROWS = ["one", "two", "three", "four", "five", "six"] as const;

function isStatusFilter(value: string): value is StatusFilter {
  return STATUS_FILTERS.some((status) => status === value);
}

export function AlertsList() {
  const workspace = useWorkspaceNavigation();
  const [status, setStatus] = useState<StatusFilter>("open");
  const [metric, setMetric] = useState<MetricFilter>("all");
  const [appId, setAppId] = useState("all");
  const query = trpc.alerts.list.useInfiniteQuery(
    {
      status,
      metric: metric === "all" ? undefined : metric,
      appId: appId === "all" ? undefined : appId,
      limit: 20,
    },
    { getNextPageParam: (lastPage) => lastPage.nextCursor },
  );
  const appOptionsQuery = trpc.alerts.list.useQuery({ status: "all", limit: 100 });
  const alerts = query.data?.pages.flatMap((page) => page.alerts) ?? [];
  const appOptions = useMemo(() => {
    const unique = new Map<string, string>();
    for (const alert of appOptionsQuery.data?.alerts ?? []) {
      unique.set(alert.appId, alert.appName);
    }
    return Array.from(unique, ([id, name]) => ({ id, name })).sort((a, b) =>
      a.name.localeCompare(b.name),
    );
  }, [appOptionsQuery.data]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col justify-between gap-3 md:flex-row md:items-center">
        <Tabs
          value={status}
          onValueChange={(value) => {
            if (isStatusFilter(value)) {
              setStatus(value);
            }
          }}
        >
          <TabsList aria-label="Alert status">
            <TabsTrigger value="open">Open</TabsTrigger>
            <TabsTrigger value="resolved">Resolved</TabsTrigger>
            <TabsTrigger value="all">All</TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Select
            value={metric}
            items={[{ value: "all", label: "All metrics" }, ...alertMetricOptions]}
            onValueChange={(value) => {
              if (value === "all" || (value !== null && isAlertMetric(value))) {
                setMetric(value);
              }
            }}
          >
            <SelectTrigger
              className="w-full bg-gray-1 sm:w-48"
              leftIcon={<Pulse iconSize="md-medium" className="text-gray-9" />}
              rightIcon={<ChevronDown iconSize="md-medium" className="absolute right-2" />}
            >
              <SelectValue placeholder="All metrics" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All metrics</SelectItem>
              {alertMetricOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={appId}
            items={[
              { value: "all", label: "All apps" },
              ...appOptions.map((app) => ({ value: app.id, label: app.name })),
            ]}
            onValueChange={(value) => value !== null && setAppId(value)}
          >
            <SelectTrigger
              className="w-full bg-gray-1 sm:w-48"
              leftIcon={<Cube iconSize="md-medium" className="text-gray-9" />}
              rightIcon={<ChevronDown iconSize="md-medium" className="absolute right-2" />}
            >
              <SelectValue placeholder="All apps" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All apps</SelectItem>
              {appOptions.map((app) => (
                <SelectItem key={app.id} value={app.id}>
                  {app.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <ResourceListContent aria-live="polite">
        {query.isLoading ? (
          <AlertsSkeleton />
        ) : query.isError ? (
          <EmptyContent title="Unable to load alerts" description={query.error.message} />
        ) : alerts.length === 0 ? (
          <EmptyContent
            title={status === "open" ? "No open alerts" : "No alerts found"}
            description="Anomaly alerts detect unusual production errors, traffic, and resource usage against each app's recent baseline."
          />
        ) : (
          <ResourceListBody aria-label="Anomaly alerts">
            {alerts.map((alert) => (
              <ResourceListItem
                key={alert.id}
                className="group flex flex-col gap-3 px-4 py-3 transition-colors hover:bg-grayA-2 lg:flex-row lg:items-center lg:gap-0"
              >
                <Link
                  href={routes.alerts.detail({
                    workspaceSlug: workspace.slug,
                    alertId: alert.id,
                  })}
                  className="absolute inset-0 z-10 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
                  aria-label={`${alertMetricLabel(alert.metric)} alert for ${alert.appName}`}
                />
                <div className="flex min-w-0 flex-col gap-1 lg:w-[23%] lg:shrink-0">
                  <span className="truncate text-[13px] font-semibold text-accent-12">
                    {alertMetricLabel(alert.metric)}
                  </span>
                  <span className="truncate text-xs text-gray-9">
                    {alert.appName} <span aria-hidden="true">›</span> {alert.environmentName}
                  </span>
                </div>
                <div className="lg:w-[13%] lg:shrink-0">
                  <AlertStatusBadge status={alert.status} />
                </div>
                <div className="flex min-w-0 flex-col gap-1 lg:w-[27%] lg:shrink-0">
                  <span className="truncate text-[13px] font-medium tabular-nums text-accent-12">
                    {hasFixedAlertThreshold(alert.metric) ? (
                      formatAlertDistance(
                        alert.metric,
                        alert.observedValue,
                        alert.baselineMean,
                        alert.baselineStddev,
                      )
                    ) : (
                      <>
                        {formatAlertValue(alert.metric, alert.observedValue)}
                        <span className="font-normal text-gray-9">
                          {" "}
                          vs {formatAlertValue(alert.metric, alert.baselineMean)} avg
                        </span>
                      </>
                    )}
                  </span>
                  {hasFixedAlertThreshold(alert.metric) ? null : (
                    <span className="text-xs font-medium tabular-nums text-error-11">
                      {formatSigma(alert.observedValue, alert.baselineMean, alert.baselineStddev)}
                    </span>
                  )}
                </div>
                <div className="relative z-20 lg:w-[14%] lg:shrink-0">
                  <TimestampInfo
                    value={alert.firedAt}
                    displayType="relative"
                    className="text-xs text-gray-9 underline decoration-dotted"
                  />
                </div>
                <div className="flex items-center gap-3 lg:w-[23%] lg:shrink-0 lg:justify-end">
                  <div className="relative z-20">
                    <AlertRowChart alertId={alert.id} metric={alert.metric} />
                  </div>
                  {alert.status === "open" ? <ResolveAlertButton alertId={alert.id} /> : null}
                </div>
              </ResourceListItem>
            ))}
          </ResourceListBody>
        )}
        {query.hasNextPage ? (
          <ResourceListFooter className="justify-center">
            <Button
              variant="outline"
              loading={query.isFetchingNextPage}
              onClick={() => query.fetchNextPage()}
            >
              Load more
            </Button>
          </ResourceListFooter>
        ) : null}
      </ResourceListContent>
    </div>
  );
}

function AlertsSkeleton() {
  return (
    <ResourceListBody aria-busy="true" aria-label="Loading alerts">
      {SKELETON_ROWS.map((key) => (
        <ResourceListItem key={key} className="flex items-center gap-6 px-4 py-4">
          <Skeleton className="h-8 w-44" />
          <Skeleton className="h-5 w-20" />
          <Skeleton className="h-8 flex-1" />
          <Skeleton className="h-7 w-[158px]" />
        </ResourceListItem>
      ))}
    </ResourceListBody>
  );
}

function EmptyContent({ title, description }: { title: string; description: string }) {
  return (
    <div className="p-3">
      <Empty className="min-h-[240px] rounded-lg border border-dashed border-gray-4 bg-gray-1/50">
        <Empty.Title>{title}</Empty.Title>
        <Empty.Description className="max-w-md">{description}</Empty.Description>
      </Empty>
    </div>
  );
}
