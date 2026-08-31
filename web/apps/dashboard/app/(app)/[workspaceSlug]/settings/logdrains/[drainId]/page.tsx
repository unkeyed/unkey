"use client";

import { trpc } from "@/lib/trpc/client";
import { Empty, PageBody, PageContainer, PageHeader, PageHeaderContent, Skeleton } from "@unkey/ui";
import { use } from "react";
import { DrainDetail } from "./drain-detail";

export default function LogdrainDetailPage(props: { params: Promise<{ drainId: string }> }) {
  const { drainId } = use(props.params);
  const query = trpc.logdrain.get.useQuery({ id: drainId });
  const metrics = trpc.logdrain.metrics.useQuery({ drainId, hours: 24 });
  const recentErrors = trpc.logdrain.recentErrors.useQuery({ drainId });
  const drain = query.data;

  if (query.isLoading) {
    return <DetailSkeleton />;
  }
  if (query.isError) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Unable to load log drain</Empty.Title>
            <Empty.Description>{query.error.message}</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }
  if (!drain) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Log drain not found</Empty.Title>
            <Empty.Description>It may have been deleted.</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }
  return (
    <DrainDetail
      drain={drain}
      metricsSeries={metrics.data?.series}
      metricsLoading={metrics.isLoading}
      metricsError={metrics.isError}
      recentErrorEntries={recentErrors.data}
      recentErrorsLoading={recentErrors.isLoading}
      recentErrorsError={recentErrors.isError}
    />
  );
}

function DetailSkeleton() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <Skeleton className="h-5 w-48" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-busy="true" aria-live="polite">
        <output className="sr-only">Loading log drain…</output>
        <Skeleton className="h-52 rounded-xl" />
        <Skeleton className="h-96 rounded-xl" />
      </PageBody>
    </PageContainer>
  );
}
