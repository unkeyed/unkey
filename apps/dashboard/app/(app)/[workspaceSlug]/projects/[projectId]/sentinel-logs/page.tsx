"use client";
import { useCallback, useState } from "react";
import { ProjectContentWrapper } from "../components/project-content-wrapper";
import { SentinelLogsChart } from "./components/charts";
import { SentinelLogsControlCloud } from "./components/control-cloud";
import { SentinelLogsControls } from "./components/controls";
import { SentinelLogDetails } from "./components/table/sentinel-log-details";
import { SentinelLogsTable } from "./components/table/sentinel-logs-table";
import { SentinelLogsProvider } from "./context/sentinel-logs-provider";

export default function Page() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <ProjectContentWrapper>
      <SentinelLogsProvider>
        <SentinelLogsControls />
        <SentinelLogsControlCloud />
        <SentinelLogsChart onMount={handleDistanceToTop} />
        <SentinelLogsTable />
        <SentinelLogDetails distanceToTop={tableDistanceToTop} />
      </SentinelLogsProvider>
    </ProjectContentWrapper>
  );
}
