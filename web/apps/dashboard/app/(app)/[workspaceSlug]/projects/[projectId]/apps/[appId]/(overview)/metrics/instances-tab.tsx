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

export function InstancesTab({ view, markers, syncId, ...handlers }: Props) {
  return (
    <div className="flex flex-col gap-4">
      {match(view)
        .with({ status: "loading" }, () => (
          <Card className="overflow-hidden">
            <div className="grid divide-y divide-grayA-4 md:grid-cols-2 md:divide-x md:divide-y-0">
              {INSTANCE_METRICS.map((metric) => (
                <div key={metric} className="p-4">
                  <Skeleton className="h-[220px] w-full rounded-md" />
                </div>
              ))}
            </div>
            <div className="flex min-h-11 items-center border-t border-grayA-4 px-3">
              <Skeleton className="h-7 w-24 rounded-md" />
            </div>
            <div className="border-t border-grayA-4 p-4">
              <Skeleton className="h-[280px] w-full rounded-md" />
            </div>
          </Card>
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
              <EmptyStateTitle>No instances in this range</EmptyStateTitle>
              <EmptyStateDescription>
                Select a longer range or deploy an app to see instance metrics.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        ))
        .with({ status: "ready" }, (ready) => (
          <InstancesLayout view={ready} markers={markers} syncId={syncId} {...handlers} />
        ))
        .exhaustive()}
    </div>
  );
}
