"use client";
import {
  createAuditLogColumns,
  getAuditRowClassName,
  getAuditSelectedClassName,
  renderAuditLogSkeletonRow,
  useAuditLogsQuery,
} from "@/components/audit-logs-table";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { IconBookBookmarkOutline18, IconInputSearchOutline18 } from "@unkey/icons";
import {
  DataTable,
  type DataTableRef,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PaginationFooter,
  buttonVariants,
} from "@unkey/ui";
import { useEffect, useMemo, useRef } from "react";

type Props = {
  selectedLog: AuditLog | null;
  setSelectedLog: (log: AuditLog | null) => void;
  onMount: (distanceToTop: number) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog, onMount }: Props) => {
  const tableRef = useRef<DataTableRef>(null);
  const {
    auditLogs,
    isLoading,
    isNavigating,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  } = useAuditLogsQuery();

  useEffect(() => {
    const distanceToTop = tableRef.current?.containerRef?.getBoundingClientRect().top ?? 0;
    onMount(distanceToTop);
  }, [onMount]);

  useEffect(() => {
    if (selectedLog && !auditLogs.some((log) => log.auditLog.id === selectedLog.auditLog.id)) {
      setSelectedLog(null);
    }
  }, [auditLogs, selectedLog, setSelectedLog]);

  const columns = useMemo(() => createAuditLogColumns({ selectedLog }), [selectedLog]);

  return (
    <>
      <DataTable
        ref={tableRef}
        data={auditLogs}
        columns={columns}
        getRowId={(log) => log.auditLog.id}
        isLoading={isLoading}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={(log) => getAuditRowClassName(log, selectedLog)}
        selectedClassName={getAuditSelectedClassName}
        renderSkeletonRow={renderAuditLogSkeletonRow}
        emptyState={
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconInputSearchOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No Audit Logs Found</EmptyStateTitle>
              <EmptyStateDescription>
                There are no audit logs matching your filters. Adjust your search criteria or check
                back later.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/audit-log/introduction"
                target="_blank"
                rel="noopener noreferrer"
                className={buttonVariants({ variant: "outline", size: "md" })}
              >
                <IconBookBookmarkOutline18 />
                Learn about Audit Logs
              </a>
            </EmptyStateActions>
          </EmptyState>
        }
        config={{
          rowHeight: 26,
          layout: "classic",
          rowBorders: true,
          loadingRows: 50,
        }}
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        disabled={isNavigating}
      />
    </>
  );
};
