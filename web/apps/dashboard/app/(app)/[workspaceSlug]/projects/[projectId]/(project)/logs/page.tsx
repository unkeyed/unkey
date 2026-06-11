"use client";

import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { RuntimeLogsView } from "./runtime-logs-view";

export default function RuntimeLogsPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <RuntimeLogsView />
    </PageContainer>
  );
}
