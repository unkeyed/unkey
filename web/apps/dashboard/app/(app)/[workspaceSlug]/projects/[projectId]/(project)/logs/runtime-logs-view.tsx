"use client";

import { useState } from "react";
import { RuntimeLogsControlCloud } from "./components/control-cloud";
import { RuntimeLogsControls } from "./components/controls";
import { RuntimeLogsCrashBanner } from "./components/runtime-logs-crash-banner";
import { RuntimeLogDetails } from "./components/table/runtime-log-details";
import { RuntimeLogsTable } from "./components/table/runtime-logs-table";
import { RuntimeLogsProvider } from "./context/runtime-logs-provider";

/**
 * The runtime logs surface: controls, table, and the detail panel. Used by the
 * project-wide logs page and the per-deployment Logs tab. The query and filters
 * key off route params, so dropping this on a deployment route scopes it to
 * that deployment (see useRuntimeLogsQuery / RuntimeLogsFilters).
 */
export function RuntimeLogsView() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <RuntimeLogsProvider>
      <RuntimeLogsControls />
      <div
        ref={(el) => {
          if (el) {
            setTableDistanceToTop(el.getBoundingClientRect().top);
          }
        }}
      />
      <RuntimeLogsControlCloud />
      <RuntimeLogsCrashBanner />
      <RuntimeLogsTable />
      <RuntimeLogDetails distanceToTop={tableDistanceToTop} />
    </RuntimeLogsProvider>
  );
}
