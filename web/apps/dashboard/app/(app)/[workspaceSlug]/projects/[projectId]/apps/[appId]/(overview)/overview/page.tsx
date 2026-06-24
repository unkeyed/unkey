"use client";

import { trpc } from "@/lib/trpc/client";
import { PageBody, PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useParams } from "next/navigation";

export default function Overview() {
  const params = useParams();
  const appId = typeof params?.appId === "string" ? params.appId : undefined;

  const { data } = trpc.deploy.metrics.getAppRpsMetrics.useQuery(
    // biome-ignore lint/style/noNonNullAssertion: We check it below
    { appId: appId! },
    { enabled: !!appId },
  );

  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Overview</PageHeaderTitle>
          {JSON.stringify(data, null, 2)}
        </PageHeaderContent>
      </PageHeader>
      <PageBody />
    </PageContainer>
  );
}
