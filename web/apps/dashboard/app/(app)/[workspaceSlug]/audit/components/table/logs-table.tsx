"use client";
import {
  EmptyAuditLogs,
  createAuditLogColumns,
  getAuditRowClassName,
  getAuditSelectedClassName,
  renderAuditLogSkeletonRow,
  useAuditLogsQuery,
} from "@/components/audit-logs-table";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { DataTable, type DataTableRef, PaginationFooter } from "@unkey/ui";
import { useEffect, useMemo, useRef } from "react";

type Props = {
  selectedLog: AuditLog | null;
  setSelectedLog: (log: AuditLog | null) => void;
  onMount: (distanceToTop: number) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog, onMount }: Props) => {
  const tableRef = useRef<DataTableRef>(null);
  const { auditLogs, isLoading, page, pageSize, totalPages, totalCount, onPageChange } =
    useAuditLogsQuery();

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
        emptyState={<EmptyAuditLogs />}
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
      />
    </>
  );
};
