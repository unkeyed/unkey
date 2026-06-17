"use client";

import { PageContainer, PageHeader, PageHeaderContent, PageHeaderTitle } from "@unkey/ui";
import { useState } from "react";
import { RuntimeLogsControlCloud } from "./components/control-cloud";
import { RuntimeLogsControls } from "./components/controls";
import { RuntimeLogDetails } from "./components/table/runtime-log-details";
import { RuntimeLogsTable } from "./components/table/runtime-logs-table";
import { RuntimeLogsProvider } from "./context/runtime-logs-provider";

export default function RuntimeLogsPage() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <PageContainer width="full">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Logs</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <RuntimeLogsProvider>
        <RuntimeLogsControls />
        <div
          ref={(el) => {
            if (el) {
              const rect = el.getBoundingClientRect();
              setTableDistanceToTop(rect.top);
            }
          }}
        />
        <RuntimeLogsControlCloud />
        {/* Railway-style summary header: hidden when there are no
            crashes in the visible window, otherwise renders a single
            line with count + last failure context. The inline banners
            below only emit on actual `terminated` events; this banner
            is the headline so users don't have to scan to spot a loop. */}
        <RuntimeLogsTable />
        <RuntimeLogDetails distanceToTop={tableDistanceToTop} />
      </RuntimeLogsProvider>
    </PageContainer>
  );
}
