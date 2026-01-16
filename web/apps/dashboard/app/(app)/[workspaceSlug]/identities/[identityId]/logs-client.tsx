"use client";

import { ErrorBoundary } from "@/components/error-boundary";
import type { IdentityLog } from "@/lib/trpc/routers/identity/query-logs";
import { Refresh3, TriangleWarning } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useCallback, useState } from "react";
import { IdentityDetailsLogsChart } from "./components/charts";
import { IdentityDetailsLogsControlCloud } from "./components/control-cloud";
import { IdentityDetailsLogsControls } from "./components/controls";
import { IdentityDetailsDrawer } from "./components/table/components/log-details";
import { IdentityDetailsLogsTable } from "./components/table/logs-table";
import { IdentityDetailsLogsProvider } from "./context/logs";

export const IdentityDetailsLogsClient = ({
  identityId,
}: {
  identityId: string;
}) => {
  const [selectedLog, setSelectedLog] = useState<IdentityLog | null>(null);
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const handleSelectedLog = useCallback((log: IdentityLog | null) => {
    setSelectedLog(log);
  }, []);

  const createErrorFallback = (componentName: string) => (error: Error, reset: () => void) => (
    <div className="flex items-center justify-center w-full h-32 bg-error-2 border border-error-6 rounded-lg">
      <div className="text-center p-4">
        <TriangleWarning className="w-8 h-8 text-error-9 mx-auto mb-2" />
        <h3 className="text-sm font-medium text-error-11 mb-1">{componentName} Error</h3>
        <p className="text-xs text-gray-11 mb-3">{error.message}</p>
        <Button variant="outline" size="sm" onClick={reset} className="text-xs">
          <Refresh3 className="w-3 h-3 mr-1" />
          Retry
        </Button>
      </div>
    </div>
  );

  return (
    <IdentityDetailsLogsProvider>
      <div className="flex flex-col">
        <ErrorBoundary fallback={createErrorFallback("Controls")}>
          <IdentityDetailsLogsControls identityId={identityId} />
        </ErrorBoundary>

        <ErrorBoundary fallback={createErrorFallback("Active Filters")}>
          <IdentityDetailsLogsControlCloud />
        </ErrorBoundary>

        <div className="flex flex-col">
          <ErrorBoundary fallback={createErrorFallback("Chart")}>
            <IdentityDetailsLogsChart identityId={identityId} onMount={handleDistanceToTop} />
          </ErrorBoundary>

          <ErrorBoundary fallback={createErrorFallback("Logs Table")}>
            <IdentityDetailsLogsTable
              selectedLog={selectedLog}
              onLogSelect={handleSelectedLog}
              identityId={identityId}
            />
          </ErrorBoundary>
        </div>

        <ErrorBoundary fallback={createErrorFallback("Log Details")}>
          <IdentityDetailsDrawer
            distanceToTop={tableDistanceToTop}
            onLogSelect={handleSelectedLog}
            selectedLog={selectedLog}
          />
        </ErrorBoundary>
      </div>
    </IdentityDetailsLogsProvider>
  );
};
