"use client";

import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { RuntimeLogsControlCloud } from "./components/control-cloud";
import { RuntimeLogsControls } from "./components/controls";
import { RuntimeLogDetails } from "./components/table/runtime-log-details";
import { RuntimeLogsTable } from "./components/table/runtime-logs-table";
import { RuntimeLogsProvider } from "./context/runtime-logs-provider";

export default function RuntimeLogsPage() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <RuntimeLogsProvider>
        <RuntimeLogsControls />
        <RuntimeLogsControlCloud />
        <RuntimeLogsTable />
        <RuntimeLogDetails />
      </RuntimeLogsProvider>
    </PageContainer>
  );
}
