"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { RANGE_PRESETS } from "@/lib/metrics/app-metrics";
import { RANGE_LABELS } from "@/lib/metrics/views";
import { match } from "@unkey/match";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import { EnvironmentSelect } from "../../../_components/environment-select";
import { ChartHoverProvider } from "./chart-hover";
import { DeployMarkerPopover } from "./deploy-marker-popover";
import { InstancesTab } from "./instances-tab";
import { MetricCard } from "./metric-card";
import { MetricPanel } from "./metric-panel";
import { METRICS_TABS, type MetricsPageModel, TABS } from "./model";
import { RegionSelect } from "./region-select";
import { ReplicaSheet } from "./replica-sheet";
import { type MetricsPageController, useMetricsPage } from "./use-metrics-page";

const SYNC_ID = "app-metrics";

type ReadyModel = Extract<MetricsPageModel, { kind: "ready" }>;

export function MetricsPage() {
  const page = useMetricsPage();

  return (
    <PageContainer>
      <Tabs
        value={page.tab}
        onValueChange={(tab) => isOneOf(METRICS_TABS, tab) && page.setTab(tab)}
        className="contents"
      >
        <PageHeader>
          <PageHeaderContent>
            <PageHeaderTitle>Metrics</PageHeaderTitle>
          </PageHeaderContent>
          <PageHeaderActions>
            <MetricsControls page={page} />
          </PageHeaderActions>
          <TabsList variant="underline">
            {METRICS_TABS.map((tab) => (
              <TabsTrigger key={tab} value={tab}>
                {TABS[tab].label}
              </TabsTrigger>
            ))}
          </TabsList>
        </PageHeader>
        <PageBody>
          <ChartHoverProvider>
            {match(page.model)
              .with({ kind: "loading" }, () => <MetricsSkeleton />)
              .with({ kind: "no-environment" }, () => (
                <EmptyState>
                  <EmptyStateHeader>
                    <EmptyStateTitle>No environments yet</EmptyStateTitle>
                    <EmptyStateDescription>
                      Metrics show up once this app has an environment with a deployment.
                    </EmptyStateDescription>
                  </EmptyStateHeader>
                </EmptyState>
              ))
              .with({ kind: "ready" }, (model) => <ReadyMetrics model={model} page={page} />)
              .exhaustive()}
          </ChartHoverProvider>
        </PageBody>
      </Tabs>
    </PageContainer>
  );
}

function ReadyMetrics({ model, page }: { model: ReadyModel; page: MetricsPageController }) {
  const markers = chartMarkers(model, page);
  return (
    <>
      <TabsContent value={model.tab} className="mt-0">
        {match(model.view)
          .with({ kind: "metrics" }, ({ hero, grid }) => (
            <div className="grid gap-4 md:grid-cols-2">
              <div className="md:col-span-2">
                <MetricCard
                  card={hero}
                  markers={markers}
                  height={240}
                  syncId={SYNC_ID}
                  onExpand={() => page.expand(hero.metric)}
                />
              </div>
              {grid.map((card) => (
                <MetricCard
                  key={card.metric}
                  card={card}
                  markers={markers}
                  height={160}
                  syncId={SYNC_ID}
                  onExpand={() => page.expand(card.metric)}
                />
              ))}
            </div>
          ))
          .with({ kind: "instances" }, ({ instances }) => (
            <InstancesTab
              view={instances}
              markers={markers}
              syncId={SYNC_ID}
              onOpen={page.openReplica}
              logsHref={page.logsHref}
              onClearInstance={page.clearInstance}
            />
          ))
          .exhaustive()}
      </TabsContent>
      <MetricPanel
        panel={model.panel}
        environmentSlug={model.environment.slug}
        range={page.range}
        markers={markers}
        onClose={page.collapse}
      />
      <ReplicaSheet sheet={page.sheet} markers={markers} onClose={page.closeReplica} />
    </>
  );
}

function chartMarkers(model: ReadyModel, page: MetricsPageController): ChartMarkers {
  return {
    items: model.markers,
    activeId: page.activeDeployId,
    onActiveChange: page.setActiveDeployId,
    renderPopover: (id) => {
      const deploy = model.deploys.get(id);
      return deploy ? <DeployMarkerPopover deploy={deploy} /> : null;
    },
  };
}

function MetricsControls({ page }: { page: MetricsPageController }) {
  const regionFilter = page.model.kind === "ready" ? page.model.regionFilter : null;
  return (
    <>
      {regionFilter && (
        <RegionSelect
          regions={regionFilter.regions}
          value={regionFilter.region}
          onChange={page.setRegion}
        />
      )}
      <EnvironmentSelect
        includeAll={false}
        value={page.environmentId ?? ""}
        onValueChange={page.setEnvironment}
        placeholder="Environment"
        size="sm"
        wrapperClassName="w-fit"
      />
      <Select
        value={page.range}
        onValueChange={(range) => isOneOf(RANGE_PRESETS, range) && page.setRange(range)}
        items={RANGE_LABELS}
      >
        <SelectTrigger size="sm" wrapperClassName="w-fit">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {RANGE_PRESETS.map((range) => (
            <SelectItem key={range} value={range}>
              {RANGE_LABELS[range]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </>
  );
}

function isOneOf<T extends string>(list: readonly T[], value: unknown): value is T {
  return list.some((item) => item === value);
}

function MetricsSkeleton() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Skeleton className="h-[310px] w-full rounded-lg md:col-span-2" />
      <Skeleton className="h-[230px] w-full rounded-lg" />
      <Skeleton className="h-[230px] w-full rounded-lg" />
    </div>
  );
}
