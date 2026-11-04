"use client";
import {
  createAuditLogColumns,
  getAuditRowClassName,
  getAuditSelectedClassName,
  renderAuditLogSkeletonRow,
  useAuditLogsQuery,
} from "@/components/audit-logs-table";
import { EmptyOrLoadError } from "@/components/load-error";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { IconBookBookmarkOutline18, IconInputSearchOutline18 } from "@unkey/icons";
import {
  DataTable,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  buttonVariants,
} from "@unkey/ui";
import { useEffect, useMemo } from "react";

type Props = {
  selectedLog: AuditLog | null;
  setSelectedLog: (log: AuditLog | null) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog }: Props) => {
  const { table, totalCount, timeWindowLabel } = useAuditLogsQuery();

  useEffect(() => {
    if (selectedLog && !table.rows.some((log) => log.auditLog.id === selectedLog.auditLog.id)) {
      setSelectedLog(null);
    }
  }, [table.rows, selectedLog, setSelectedLog]);

  const columns = useMemo(() => createAuditLogColumns({ selectedLog }), [selectedLog]);

  return (
    <>
      <DataTable
        data={table.rows}
        columns={columns}
        getRowId={(log) => log.auditLog.id}
        isLoading={table.isLoading}
        loadMore={{
          singular: "event",
          plural: "events",
          scope: timeWindowLabel,
          state: table.footer,
          totalCount,
        }}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={(log) => getAuditRowClassName(log, selectedLog)}
        selectedClassName={getAuditSelectedClassName}
        renderSkeletonRow={renderAuditLogSkeletonRow}
        emptyState={
          <EmptyOrLoadError failure={table.failure} title="Could not load audit logs">
            <EmptyState frame="none">
              <EmptyStateIcon>
                <IconInputSearchOutline18 />
              </EmptyStateIcon>
              <EmptyStateHeader>
                <EmptyStateTitle>No Audit Logs Found</EmptyStateTitle>
                <EmptyStateDescription>
                  There are no audit logs matching your filters. Adjust your search criteria or
                  check back later.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>
                <a
                  href="https://www.unkey.com/docs/audit-log/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={buttonVariants({ variant: "outline", size: "sm" })}
                >
                  <IconBookBookmarkOutline18 />
                  Learn about Audit Logs
                </a>
              </EmptyStateActions>
            </EmptyState>
          </EmptyOrLoadError>
        }
        config={{
          rowHeight: 26,
          layout: "classic",
          rowBorders: true,
          loadingRows: 50,
        }}
      />
    </>
  );
};
