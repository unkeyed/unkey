"use client";

import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { SentinelLogsView } from "./sentinel-logs-view";

export default function Page() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SentinelLogsView />
    </PageContainer>
  );
}
