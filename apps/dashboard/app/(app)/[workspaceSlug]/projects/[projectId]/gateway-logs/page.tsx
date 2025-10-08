"use client";
import { useCallback, useState } from "react";
import { ProjectContentWrapper } from "../components/project-content-wrapper";
import { GatewayLogsChart } from "./components/charts";
import { GatewayLogsControlCloud } from "./components/control-cloud";
import { GatewayLogsControls } from "./components/controls";
import { GatewayLogDetails } from "./components/table/gateway-log-details";
import { GatewayLogsTable } from "./components/table/gateway-logs-table";
import { GatewayLogsProvider } from "./context/gateway-logs-provider";

export default function Page() {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <ProjectContentWrapper>
      <GatewayLogsProvider>
        <GatewayLogsControls />
        <GatewayLogsControlCloud />
        <GatewayLogsChart onMount={handleDistanceToTop} />
        <GatewayLogsTable />
        <GatewayLogDetails distanceToTop={tableDistanceToTop} />
      </GatewayLogsProvider>
    </ProjectContentWrapper>
  );
}
