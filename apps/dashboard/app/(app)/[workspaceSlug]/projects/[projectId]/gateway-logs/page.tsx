"use client";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useState } from "react";
import { useProjectLayout } from "../layout-provider";
import { GatewayLogsChart } from "./components/charts";
import { GatewayLogsControlCloud } from "./components/control-cloud";
import { GatewayLogsControls } from "./components/controls";
import { GatewayLogDetails } from "./components/table/gateway-log-details";
import { GatewayLogsTable } from "./components/table/gateway-logs-table";
import { GatewayLogsProvider } from "./context/gateway-logs-provider";

export default function Page() {
  const { isDetailsOpen } = useProjectLayout();
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <div
      className={cn(
        "flex flex-col transition-all duration-300 ease-in-out",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
      )}
    >
      <GatewayLogsProvider>
        <GatewayLogsControls />
        <GatewayLogsControlCloud />
        <GatewayLogsChart onMount={handleDistanceToTop} />
        <GatewayLogsTable />
        <GatewayLogDetails distanceToTop={tableDistanceToTop} />
      </GatewayLogsProvider>
    </div>
  );
}
