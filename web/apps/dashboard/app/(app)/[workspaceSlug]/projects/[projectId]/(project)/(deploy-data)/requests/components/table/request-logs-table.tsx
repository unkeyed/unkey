"use client";
import { EmptyOrLoadError } from "@/components/load-error";

import {
  createRequestLogsColumns,
  getRowClassName,
  getSelectedClassName,
  useRequestLogsQuery,
} from "@/components/request-logs-table";
import { IconArrowsOppositeDirectionYOutline18, IconBookBookmarkOutline18 } from "@unkey/icons";
import {
  Button,
  DataTable,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { useMemo } from "react";
import { useRequestLogsContext } from "../../context/request-logs-provider";

export const RequestLogsTable = () => {
  const { setSelectedLog, selectedLog, isLive, refreshNonce } = useRequestLogsContext();
  const { realtimeLogs, table, totalCount } = useRequestLogsQuery({
    startPolling: isLive,
    pollIntervalMs: 2000,
    refreshNonce,
  });

  const columns = useMemo(() => createRequestLogsColumns(), []);

  return (
    <div className="flex flex-col md:flex-1">
      <DataTable
        data={table.rows}
        realtimeData={realtimeLogs}
        columns={columns}
        getRowId={(log) => log.request_id}
        isLoading={table.isLoading}
        loadMore={{
          singular: "request",
          plural: "requests",
          state: table.footer,
          totalCount,
        }}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={(log) => getRowClassName(log, selectedLog, isLive, realtimeLogs)}
        selectedClassName={getSelectedClassName}
        emptyState={
          <EmptyOrLoadError failure={table.failure} title="Could not load requests">
            <EmptyState frame="none">
              <EmptyStateIcon>
                <IconArrowsOppositeDirectionYOutline18 />
              </EmptyStateIcon>
              <EmptyStateHeader>
                <EmptyStateTitle>Logs</EmptyStateTitle>
                <EmptyStateDescription>
                  Keep track of all activity within your workspace. We collect all API requests,
                  giving you a clear history to find problems or debug issues.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button variant="outline" size="sm">
                    <IconBookBookmarkOutline18 />
                    Documentation
                  </Button>
                </a>
              </EmptyStateActions>
            </EmptyState>
          </EmptyOrLoadError>
        }
        // VirtualTable defaulted to 26px dense rows and 50 loading rows; DataTable's
        // defaults (36px / 10) differ, so set them explicitly to preserve the layout.
        config={{
          rowHeight: 26,
          layout: "classic",
          rowBorders: false,
          loadingRows: 50,
          striped: true,
        }}
      />
    </div>
  );
};
