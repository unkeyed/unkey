"use client";
import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useState } from "react";
import { RequestLogsControlCloud } from "./components/control-cloud";
import { RequestLogsControls } from "./components/controls";
import { RequestLogDetails } from "./components/table/request-log-details";
import { RequestLogsTable } from "./components/table/request-logs-table";
import { RequestLogsProvider } from "./context/request-logs-provider";

export default function Page() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <RequestLogsProvider>
        <RequestLogsControls />
        <div
          ref={(el) => {
            if (el) {
              const rect = el.getBoundingClientRect();
              setTableDistanceToTop(rect.top);
            }
          }}
        />
        <RequestLogsControlCloud />
        <RequestLogsTable />
        <RequestLogDetails distanceToTop={tableDistanceToTop} />
      </RequestLogsProvider>
    </PageContainer>
  );
}
