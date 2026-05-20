"use client";

import { useSort } from "@/components/logs/hooks/use-sort";
import {
  createRatelimitsOverviewColumns,
  getRowClassName,
  renderRatelimitsOverviewSkeletonRow,
  useRatelimitsOverviewListPaginated,
} from "@/components/ratelimits-overview-table";
import type { RowSelectionState, SortingState } from "@tanstack/react-table";
import type { RatelimitOverviewLog } from "@unkey/clickhouse/src/ratelimits";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, type DataTableConfig, Empty, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useRef, useState } from "react";
import { type SortFields, sortFields } from "./query-logs.schema";

const TABLE_CONFIG: Partial<DataTableConfig> = {
  rowHeight: 26,
  layout: "grid",
  rowBorders: true,
  containerPadding: "px-0",
  loadingRows: 10,
};

export const RatelimitOverviewLogsTable = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const [selectedLog, setSelectedLog] = useState<RatelimitOverviewLog | null>(null);
  const { sorts, setSorts } = useSort<SortFields>();

  const {
    historicalLogs,
    isLoading,
    isNavigating,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
  } = useRatelimitsOverviewListPaginated({ namespaceId });

  const columns = useMemo(() => createRatelimitsOverviewColumns({ namespaceId }), [namespaceId]);

  // The TanStack column id "time" maps to the server sort field "time"; for
  // the other sortable columns they happen to share names.
  const sorting: SortingState = useMemo(
    () =>
      sorts.length > 0
        ? sorts.map((s) => ({ id: s.column, desc: s.direction === "desc" }))
        : [{ id: "time", desc: true }],
    [sorts],
  );

  const sortingRef = useRef(sorting);
  sortingRef.current = sorting;

  const handleSortingChange = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sortingRef.current) : updater;
      const validated = next.flatMap((s) => {
        const result = sortFields.safeParse(s.id);
        if (!result.success) {
          return [];
        }
        return [{ column: result.data, direction: s.desc ? ("desc" as const) : ("asc" as const) }];
      });
      setSorts(validated);
    },
    [setSorts],
  );

  const rowSelection = useMemo<RowSelectionState>(
    () => (selectedLog ? { [selectedLog.identifier]: true } : {}),
    [selectedLog],
  );

  const rowClassNameMemoized = useCallback(
    (log: RatelimitOverviewLog) => getRowClassName(log, selectedLog),
    [selectedLog],
  );

  return (
    <div className="flex flex-col">
      <DataTable
        data={historicalLogs}
        isLoading={isLoading}
        columns={columns}
        getRowId={(log) => log.identifier}
        onRowClick={setSelectedLog}
        selectedItem={selectedLog}
        rowClassName={rowClassNameMemoized}
        sorting={sorting}
        onSortingChange={handleSortingChange}
        manualSorting={true}
        enableRowSelection={true}
        rowSelection={rowSelection}
        config={TABLE_CONFIG}
        renderSkeletonRow={renderRatelimitsOverviewSkeletonRow}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>Logs</Empty.Title>
              <Empty.Description className="text-left">
                No rate limit data to show. Once requests are made, you'll see a summary of passed
                and blocked requests for each rate limit identifier.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Documentation
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        }
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="rate limit identifiers"
        loading={isLoading}
        disabled={isNavigating}
      />
    </div>
  );
};
