"use client";

import type { ChartMarkers } from "@/components/charts/chart-markers";
import { IconServerOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Card,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  Skeleton,
} from "@unkey/ui";
import { InstancesLayout, type LayoutHandlers } from "./instances-layout";
import { INSTANCE_METRICS, type InstancesTabView } from "./instances-model";

type Props = LayoutHandlers & {
  view: InstancesTabView;
  markers: ChartMarkers;
  syncId: string;
};

export function InstancesTab({ view, markers, syncId, onOpen, logsHref, onClearInstance }: Props) {
  return (
    <div className="flex flex-col gap-4">
      {match(view)
        .with({ status: "loading" }, () => (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              {INSTANCE_METRICS.map((metric) => (
                <Skeleton key={metric} className="h-[260px] w-full rounded-lg" />
              ))}
            </div>
            <Skeleton className="h-[320px] w-full rounded-lg" />
          </>
        ))
        .with({ status: "failed" }, () => (
          <Card className="flex h-40 items-center justify-center text-xs text-gray-10">
            Instances are unavailable right now
          </Card>
        ))
        .with({ status: "empty" }, () => (
          <EmptyState>
            <EmptyStateIcon>
              <IconServerOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No running instances</EmptyStateTitle>
              <EmptyStateDescription>
                Instances show up here once a deployment is running in this environment.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        ))
        .with({ status: "ready" }, (ready) => (
          <InstancesLayout
            view={ready}
            markers={markers}
            syncId={syncId}
            onOpen={onOpen}
            logsHref={logsHref}
            onClearInstance={onClearInstance}
          />
        ))
        .exhaustive()}
    </div>
  );
}
