"use client";
import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useState } from "react";
import { SentinelLogsControlCloud } from "./components/control-cloud";
import { SentinelLogsControls } from "./components/controls";
import { SentinelLogDetails } from "./components/table/sentinel-log-details";
import { SentinelLogsTable } from "./components/table/sentinel-logs-table";
import { SentinelLogsProvider } from "./context/sentinel-logs-provider";

export default function Page() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Requests</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <SentinelLogsProvider>
        <SentinelLogsControls />
        <div
          ref={(el) => {
            if (el) {
              const rect = el.getBoundingClientRect();
              setTableDistanceToTop(rect.top);
            }
          }}
        />
        <SentinelLogsControlCloud />
        <SentinelLogsTable />
        <SentinelLogDetails distanceToTop={tableDistanceToTop} />
      </SentinelLogsProvider>
    </PageContainer>
  );
}
