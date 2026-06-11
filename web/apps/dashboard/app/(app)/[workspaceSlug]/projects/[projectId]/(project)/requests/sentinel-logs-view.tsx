"use client";

import { useState } from "react";
import { SentinelLogsControlCloud } from "./components/control-cloud";
import { SentinelLogsControls } from "./components/controls";
import { SentinelLogDetails } from "./components/table/sentinel-log-details";
import { SentinelLogsTable } from "./components/table/sentinel-logs-table";
import { SentinelLogsProvider } from "./context/sentinel-logs-provider";

/**
 * The request (sentinel) logs surface: controls, table, and the detail panel.
 * Used by the project requests page and the per-deployment Requests tab. The
 * query and filters key off route params, so dropping this on a deployment
 * route scopes it to that deployment (see useSentinelLogsQuery / the filters).
 */
export function SentinelLogsView() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <SentinelLogsProvider>
      <SentinelLogsControls />
      <div
        ref={(el) => {
          if (el) {
            setTableDistanceToTop(el.getBoundingClientRect().top);
          }
        }}
      />
      <SentinelLogsControlCloud />
      <SentinelLogsTable />
      <SentinelLogDetails distanceToTop={tableDistanceToTop} />
    </SentinelLogsProvider>
  );
}
