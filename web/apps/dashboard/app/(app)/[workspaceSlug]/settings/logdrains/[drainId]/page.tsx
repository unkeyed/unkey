"use client";

import { trpc } from "@/lib/trpc/client";
import {
  Button,
  Card,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  Skeleton,
} from "@unkey/ui";
import { use } from "react";
import { LogdrainDetail } from "./logdrain-detail";

export default function LogdrainDetailPage(props: { params: Promise<{ drainId: string }> }) {
  const { drainId } = use(props.params);
  const utils = trpc.useUtils();

  const cached = utils.logdrain.list.getData()?.find((drain) => drain.id === drainId);
  const query = trpc.logdrain.get.useQuery(
    { id: drainId },
    { initialData: cached, initialDataUpdatedAt: 0 },
  );

  if (query.isLoading) {
    return <DetailSkeleton />;
  }

  if (query.isError && query.error.data?.code === "NOT_FOUND") {
    return <NotFound />;
  }

  if (query.isError) {
    return (
      <PageContainer>
        <PageBody>
          <Empty>
            <Empty.Title>We couldn't load this log drain</Empty.Title>
            <Empty.Description>Something went wrong on our side. Try again.</Empty.Description>
            <Empty.Actions>
              <Button variant="outline" onClick={() => query.refetch()}>
                Retry
              </Button>
            </Empty.Actions>
          </Empty>
        </PageBody>
      </PageContainer>
    );
  }

  if (!query.data) {
    return <NotFound />;
  }

  return <LogdrainDetail drain={query.data} />;
}

function NotFound() {
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

function DetailSkeleton() {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-6 w-56" />
        </PageHeaderContent>
      </PageHeader>
      <PageBody aria-busy="true" aria-live="polite">
        <output className="sr-only">Loading log drain…</output>
        <div aria-hidden="true" className="flex flex-col gap-2">
          <Skeleton className="h-4 w-24" />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: skeleton cards are static and never reorder
              <Card key={index} className="flex flex-col gap-2 p-4">
                <Skeleton className="h-3.5 w-20" />
                <Skeleton className="h-6 w-14" />
              </Card>
            ))}
          </div>
        </div>
        <Skeleton aria-hidden="true" className="h-64 w-full rounded-lg" />
      </PageBody>
    </PageContainer>
  );
}
