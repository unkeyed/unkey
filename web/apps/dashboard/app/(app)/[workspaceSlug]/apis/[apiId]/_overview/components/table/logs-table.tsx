"use client";
import { createApiRequestColumns } from "@/components/api-requests-table/columns/create-api-request-columns";
import { useKeysOverviewLogsQuery } from "@/components/api-requests-table/hooks/use-keys-overview-query";
import type { SortFields } from "@/components/api-requests-table/schema/keys-overview.schema";
import { getRowClassName } from "@/components/api-requests-table/utils/get-row-class";
import { useSort } from "@/components/logs/hooks/use-sort";
import type { RowSelectionState, SortingState } from "@tanstack/react-table";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { DataTable, type DataTableConfig, EmptyApiRequests } from "@unkey/ui";
import { useCallback, useMemo } from "react";

const TABLE_CONFIG: DataTableConfig = {
  rowHeight: 26, // compact rows, default is 36
  rowSpacing: 4,
  headerHeight: 40,
  layout: "classic" as const,
  rowBorders: false,
  containerPadding: "px-2",
  tableLayout: "fixed",
  loadingRows: 10,
};

type Props = {
  log: KeysOverviewLog | null;
  setSelectedLog: (data: KeysOverviewLog | null) => void;
  apiId: string;
};

export const KeysOverviewLogsTable = ({ apiId, setSelectedLog, log: selectedLog }: Props) => {
  const { sorts, setSorts } = useSort<SortFields>();
  const { historicalLogs, isLoading, hasMore } = useKeysOverviewLogsQuery({ apiId });

  const handleNavigate = useCallback(() => setSelectedLog(null), [setSelectedLog]);

  const columns = useMemo(
    () => createApiRequestColumns({ apiId, onNavigate: handleNavigate }),
    [apiId, handleNavigate],
  );

  const rowSelection = useMemo<RowSelectionState>(
    () => (selectedLog ? { [selectedLog.request_id]: true } : {}),
    [selectedLog],
  );

  const sorting: SortingState = useMemo(
    () => sorts.map((s) => ({ id: s.column, desc: s.direction === "desc" })),
    [sorts],
  );

  const handleSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      setSorts(
        next.map((s) => ({ column: s.id as SortFields, direction: s.desc ? "desc" : "asc" })),
      );
    },
    [sorting, setSorts],
  );

  const getRowClassNameMemoized = useCallback(
    (log: KeysOverviewLog) => getRowClassName(log, selectedLog as KeysOverviewLog),
    [selectedLog],
  );

  return (
    <DataTable
      data={historicalLogs}
      isLoading={isLoading}
      columns={columns}
      getRowId={(log) => log.request_id}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      rowClassName={getRowClassNameMemoized}
      sorting={sorting}
      onSortingChange={handleSortingChange}
      manualSorting={true}
      enableRowSelection={true}
      rowSelection={rowSelection}
      config={TABLE_CONFIG}
      loadMoreFooterProps={{
        hide: true,
        hasMore: hasMore ?? false,
      }}
      emptyState={<EmptyApiRequests />}
    />
  );
};
