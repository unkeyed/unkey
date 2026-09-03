"use client";

import { trpc } from "@/lib/trpc/client";
import { Empty, PageBody, PageContainer, PageHeader, PageHeaderContent, Skeleton } from "@unkey/ui";
import { use } from "react";
import { AlertDetail } from "./alert-detail";

export default function AlertDetailPage(props: { params: Promise<{ alertId: string }> }) {
  const { alertId } = use(props.params);
  const alert = trpc.alerts.get.useQuery({ alertId });
  const timeseries = trpc.alerts.timeseries.useQuery({ alertId });

  if (alert.isLoading) {
    return <DetailSkeleton />;
  }
  if (alert.isError) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Unable to load alert</Empty.Title>
            <Empty.Description>{alert.error.message}</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }
  if (!alert.data) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>Alert not found</Empty.Title>
            <Empty.Description>It may no longer be available.</Empty.Description>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }

  return (
    <AlertDetail
      alert={alert.data}
      timeseries={timeseries.data}
      timeseriesLoading={timeseries.isLoading}
      timeseriesError={timeseries.isError}
    />
  );
}

function DetailSkeleton() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-4 w-36" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-busy="true" aria-label="Loading alert">
        <Skeleton className="h-[430px] rounded-lg" />
        <Skeleton className="h-52 rounded-lg" />
      </PageBody>
    </PageContainer>
  );
}
