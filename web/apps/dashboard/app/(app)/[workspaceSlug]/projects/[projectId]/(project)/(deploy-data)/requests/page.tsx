"use client";
import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { RequestLogsControlCloud } from "./components/control-cloud";
import { RequestLogsControls } from "./components/controls";
import { RequestLogDetails } from "./components/table/request-log-details";
import { RequestLogsTable } from "./components/table/request-logs-table";
import { RequestLogsProvider } from "./context/request-logs-provider";

export default function Page() {
  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <RequestLogsProvider>
        <RequestLogsControls />
        <RequestLogsControlCloud />
        <RequestLogsTable />
        <RequestLogDetails />
      </RequestLogsProvider>
    </PageContainer>
  );
}
