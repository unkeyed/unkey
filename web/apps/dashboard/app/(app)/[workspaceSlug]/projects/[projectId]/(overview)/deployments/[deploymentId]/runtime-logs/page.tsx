"use client";

import { useState } from "react";
import { RuntimeLogsProvider } from "./context/runtime-logs-provider";
import { RuntimeLogsControls } from "./components/controls";
import { RuntimeLogsControlCloud } from "./components/control-cloud";
import { RuntimeLogsTable } from "./components/table/runtime-logs-table";
import { RuntimeLogDetails } from "./components/table/runtime-log-details";

export default function RuntimeLogsPage() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
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
      <RuntimeLogsTable />
      <RuntimeLogDetails distanceToTop={tableDistanceToTop} />
    </RuntimeLogsProvider>
  );
}
