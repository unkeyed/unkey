"use client";

import { AlertRow } from "@/components/alerts/alert-row";
import { alertMetricOptions, isAlertMetric } from "@/components/alerts/format";
import type { AlertMetric } from "@/components/alerts/types";
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
} from "@unkey/ui";
import { useMemo, useState } from "react";

type MetricFilter = AlertMetric | "all";

const SKELETON_ROWS = ["one", "two", "three", "four", "five", "six"] as const;

export function AlertsList() {
  const workspace = useWorkspaceNavigation();
  const [metric, setMetric] = useState<MetricFilter>("all");
  const [appId, setAppId] = useState("all");
  const query = trpc.alerts.list.useInfiniteQuery(
    {
      metric: metric === "all" ? undefined : metric,
      appId: appId === "all" ? undefined : appId,
      limit: 20,
    },
    { getNextPageParam: (lastPage) => lastPage.nextCursor },
  );
  const appOptionsQuery = trpc.alerts.list.useQuery({ limit: 100 });
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
      <div className="flex justify-end">
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
            title="No open alerts"
            description="Anomaly alerts detect unusual production errors, traffic, and resource usage against each app's recent baseline."
          />
        ) : (
          <ResourceListBody aria-label="Anomaly alerts">
            {alerts.map((alert) => (
              <AlertRow
                key={alert.id}
                alert={alert}
                href={routes.projects.apps.anomalies({
                  workspaceSlug: workspace.slug,
                  projectId: alert.projectId,
                  appId: alert.appId,
                  environmentId: alert.environmentId,
                  alertId: alert.id,
                })}
              />
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
