"use client";

import { AlertFocusCard } from "@/components/alerts/alert-focus-card";
import { AlertRow } from "@/components/alerts/alert-row";
import {
  alertSeriesMetricOptions,
  isAlertSeriesMetric,
  seriesMetricForAlert,
} from "@/components/alerts/format";
import type { AlertSeriesMetric } from "@/components/alerts/types";
import { Switch } from "@/components/ui/switch";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { trpc } from "@/lib/trpc/client";
import { ChevronDown, Layers3 } from "@unkey/icons";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListHeader,
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
} from "@unkey/ui";
import { parseAsString, useQueryStates } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { useAppId, useProjectData } from "../data-provider";
import { AnomalyChart } from "./anomaly-chart";
import { AnomalyOverview } from "./anomaly-overview";
import {
  type AnomalyRangePreset,
  dayMs,
  focusedAlertRange,
  formatRangeDateTime,
  hourMs,
  lastClosedBucketEnd,
  presetRange,
} from "./range";

const rangeOptions: ReadonlyArray<{ value: AnomalyRangePreset; label: string }> = [
  { value: "24h", label: "Last 24 hours" },
  { value: "6h", label: "Last 6 hours" },
  { value: "7d", label: "Last 7 days" },
];
const skeletonRows = ["one", "two", "three"] as const;

function isRangePreset(value: string): value is AnomalyRangePreset {
  return rangeOptions.some((option) => option.value === value);
}

export default function AnomaliesPage() {
  const appId = useAppId();
  const { environments, isEnvironmentsLoading } = useProjectData();
  const [{ env: environmentParam, alert: alertId }, setQuery] = useQueryStates({
    env: parseAsString,
    alert: parseAsString,
  });
  const activeEnvironment =
    environments.find((environment) => environment.id === environmentParam) ??
    environments.find((environment) => environment.kind === ENVIRONMENT_KIND.production) ??
    environments.at(0);
  const [metric, setMetric] = useState<AlertSeriesMetric>("requests");
  const [preset, setPreset] = useState<AnomalyRangePreset>("24h");
  const [range, setRange] = useState(() => presetRange("24h", Date.now()));
  const [overviewRange] = useState(() => {
    const endMs = lastClosedBucketEnd(Date.now(), hourMs);
    return { startMs: endMs - 7 * dayMs, endMs };
  });
  const [showDeployments, setShowDeployments] = useState(true);
  const [zoomed, setZoomed] = useState(false);
  const [selectedAlertId, setSelectedAlertId] = useState<string | null>(alertId);
  const appliedFocus = useRef<string | null>(null);

  const focusQuery = trpc.alerts.get.useQuery(
    { alertId: alertId ?? "" },
    { enabled: Boolean(alertId), staleTime: 30_000 },
  );
  const focusedAlert =
    focusQuery.data?.appId === appId && focusQuery.data.environmentId === activeEnvironment?.id
      ? focusQuery.data
      : undefined;

  useEffect(() => {
    if (!focusedAlert || appliedFocus.current === focusedAlert.id) {
      return;
    }
    appliedFocus.current = focusedAlert.id;
    setMetric(seriesMetricForAlert(focusedAlert.metric));
    setRange(focusedAlertRange(focusedAlert.windowStart, focusedAlert.windowEnd, Date.now()));
    setSelectedAlertId(focusedAlert.id);
    setZoomed(true);
  }, [focusedAlert]);

  const queryScope = activeEnvironment
    ? { appId, environmentId: activeEnvironment.id }
    : { appId, environmentId: "" };
  const seriesQuery = trpc.alerts.series.useQuery(
    { ...queryScope, metric, ...range },
    { enabled: Boolean(activeEnvironment), staleTime: 30_000 },
  );
  const overviewQuery = trpc.alerts.series.useQuery(
    { ...queryScope, metric, resolution: "1h", ...overviewRange },
    { enabled: Boolean(activeEnvironment), staleTime: 60_000 },
  );
  const alertsQuery = trpc.alerts.list.useQuery(
    { ...queryScope, ...range, limit: 100 },
    { enabled: Boolean(activeEnvironment), staleTime: 30_000 },
  );
  const deploymentsQuery = trpc.alerts.deployments.useQuery(
    { ...queryScope, ...range },
    { enabled: Boolean(activeEnvironment), staleTime: 60_000 },
  );
  const alerts = alertsQuery.data?.alerts ?? [];
  const chartAlerts =
    focusedAlert && alerts.every((alert) => alert.id !== focusedAlert.id)
      ? [...alerts, focusedAlert]
      : alerts;

  if (isEnvironmentsLoading || !activeEnvironment) {
    return <AnomaliesPageSkeleton />;
  }

  const updatePreset = (nextPreset: AnomalyRangePreset) => {
    setPreset(nextPreset);
    setRange(presetRange(nextPreset, Date.now()));
    setSelectedAlertId(null);
    setZoomed(false);
  };

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Anomalies</PageHeaderTitle>
          <PageHeaderDescription>
            Compare production behavior with the expected 24-hour baseline.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Select
            value={activeEnvironment.id}
            items={environments.map((environment) => ({
              value: environment.id,
              label: capitalize(environment.slug),
            }))}
            onValueChange={(value) => {
              if (value) {
                appliedFocus.current = null;
                setSelectedAlertId(null);
                void setQuery({ env: value, alert: null });
              }
            }}
          >
            <SelectTrigger
              className="h-9 min-w-44 bg-gray-1"
              leftIcon={<Layers3 iconSize="md-medium" className="text-gray-9" />}
              rightIcon={<ChevronDown iconSize="md-medium" className="absolute right-2" />}
            >
              <SelectValue placeholder="Environment" />
            </SelectTrigger>
            <SelectContent>
              {environments.map((environment) => (
                <SelectItem key={environment.id} value={environment.id}>
                  {capitalize(environment.slug)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>
        <section className="flex flex-col gap-3 rounded-lg border border-grayA-4 bg-gray-1 p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold text-gray-12">Seven-day overview</h2>
              <p className="text-xs text-gray-9">Select a day or choose a recent range.</p>
            </div>
            <Select
              value={preset}
              items={rangeOptions}
              onValueChange={(value) => {
                if (value && isRangePreset(value)) {
                  updatePreset(value);
                }
              }}
            >
              <SelectTrigger
                className="h-9 w-44 bg-gray-1"
                rightIcon={<ChevronDown iconSize="md-medium" className="absolute right-2" />}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {rangeOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <AnomalyOverview
            buckets={overviewQuery.data?.buckets ?? []}
            selectedRange={range}
            loading={overviewQuery.isLoading}
            onSelectRange={(nextRange) => {
              setRange({
                startMs: nextRange.startMs,
                endMs: Math.min(nextRange.endMs, lastClosedBucketEnd(Date.now())),
              });
              setZoomed(true);
              setSelectedAlertId(null);
            }}
          />
        </section>

        <section className="flex flex-col gap-3">
          <div className="flex flex-col justify-between gap-3 xl:flex-row xl:items-center">
            <Tabs
              value={metric}
              onValueChange={(value) => {
                if (isAlertSeriesMetric(value)) {
                  setMetric(value);
                  setSelectedAlertId(null);
                }
              }}
            >
              <TabsList className="h-auto flex-wrap justify-start" aria-label="Anomaly metric">
                {alertSeriesMetricOptions.map((option) => (
                  <TabsTrigger key={option.value} value={option.value} className="text-xs">
                    {option.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
            <div className="flex flex-wrap items-center gap-3">
              <span className="flex items-center gap-2 text-xs text-gray-10">
                <Switch
                  aria-label="Show deployments"
                  size="sm"
                  checked={showDeployments}
                  onCheckedChange={setShowDeployments}
                />
                Show deployments
              </span>
              {zoomed ? (
                <Button variant="outline" size="sm" onClick={() => updatePreset(preset)}>
                  Reset zoom
                </Button>
              ) : null}
            </div>
          </div>
          <AnomalyChart
            metric={metric}
            data={seriesQuery.data}
            alerts={chartAlerts}
            deployments={deploymentsQuery.data ?? []}
            selectedAlertId={selectedAlertId}
            showDeployments={showDeployments}
            loading={seriesQuery.isLoading}
            error={seriesQuery.isError}
            onZoom={(nextRange) => {
              setRange(nextRange);
              setZoomed(true);
            }}
          />
        </section>

        {focusedAlert ? <AlertFocusCard alert={focusedAlert} /> : null}

        <ResourceList>
          <ResourceListHeader>
            <div>
              <h2 className="text-sm font-medium text-gray-12">
                Anomalies between {formatRangeDateTime(range.startMs)} and{" "}
                {formatRangeDateTime(range.endMs)}
              </h2>
              <p className="mt-0.5 text-xs text-gray-9">
                All metrics for {activeEnvironment.slug}.
              </p>
            </div>
          </ResourceListHeader>
          <ResourceListContent>
            {alertsQuery.isLoading ? (
              <ResourceListBody aria-label="Loading anomalies">
                {skeletonRows.map((key) => (
                  <ResourceListItem key={key} className="flex items-center gap-6 px-4 py-4">
                    <Skeleton className="h-8 w-44" />
                    <Skeleton className="h-5 w-20" />
                    <Skeleton className="h-8 flex-1" />
                  </ResourceListItem>
                ))}
              </ResourceListBody>
            ) : alerts.length === 0 ? (
              <div className="p-3">
                <Empty className="min-h-[220px] rounded-lg border border-dashed border-gray-4 bg-gray-1/50">
                  <Empty.Title>No anomalies found</Empty.Title>
                  <Empty.Description>
                    Nothing deviated from the expected baseline during this time.
                  </Empty.Description>
                </Empty>
              </div>
            ) : (
              <ResourceListBody aria-label="Anomalies in selected range">
                {alerts.map((item) => (
                  <AlertRow
                    key={item.id}
                    alert={item}
                    selected={item.id === selectedAlertId}
                    onSelect={() => {
                      setMetric(seriesMetricForAlert(item.metric));
                      setSelectedAlertId(item.id);
                    }}
                  />
                ))}
              </ResourceListBody>
            )}
          </ResourceListContent>
        </ResourceList>
      </PageBody>
    </PageContainer>
  );
}

function AnomaliesPageSkeleton() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <Skeleton className="h-7 w-36" />
          <Skeleton className="h-4 w-72" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-label="Loading anomalies" aria-busy="true">
        <Skeleton className="h-36 rounded-lg" />
        <Skeleton className="h-[430px] rounded-lg" />
        <Skeleton className="h-64 rounded-lg" />
      </PageBody>
    </PageContainer>
  );
}

function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
