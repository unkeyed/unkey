"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { DataTable, EmptyRootKeys, PaginationFooter, getSelectableRowClassName } from "@unkey/ui";
import { useCallback, useMemo } from "react";
import { createRootKeyColumns } from "./create-root-key-columns";
import { renderRootKeySkeletonRow } from "./render-root-key-skeleton-row";

const TABLE_CONFIG = {
  loadingRows: 5,
  rowHeight: 52,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

type RootKeysDataTableProps = {
  selectedKeyId: string | null;
  onEditKey: (rootKey: RootKey) => void;
  list: {
    rootKeys: RootKey[];
    isInitialLoading: boolean;
    isNavigating: boolean;
    totalCount: number;
    onPageChange: (page: number) => void;
    page: number;
    pageSize: number;
    totalPages: number;
  };
  transport: "legacy" | "v2";
};

export function RootKeysDataTable({
  selectedKeyId,
  onEditKey,
  list,
  transport,
}: RootKeysDataTableProps) {
  const {
    rootKeys,
    isInitialLoading,
    isNavigating,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
  } = list;

  const columns = useMemo(
    () => createRootKeyColumns({ onEditKey, transport }),
    [onEditKey, transport],
  );

  const selectedRootKey = useMemo(
    () => rootKeys.find((rootKey) => rootKey.id === selectedKeyId) ?? null,
    [rootKeys, selectedKeyId],
  );

  const handleRowClick = useCallback(
    (rootKey: RootKey | null) => {
      if (rootKey) {
        onEditKey(rootKey);
      }
    },
    [onEditKey],
  );

  return (
    <>
      <div className="w-full" aria-busy={isInitialLoading}>
        {isInitialLoading ? <output className="sr-only">Loading Root Keys...</output> : null}
        <DataTable
          data={rootKeys}
          columns={columns}
          getRowId={(rootKey) => rootKey.id}
          isLoading={isInitialLoading}
          onRowClick={handleRowClick}
          selectedItem={selectedRootKey}
          rowClassName={(rootKey) => getSelectableRowClassName(rootKey.id === selectedKeyId)}
          emptyState={<EmptyRootKeys />}
          config={TABLE_CONFIG}
          renderSkeletonRow={renderRootKeySkeletonRow}
        />
      </div>
      <PaginationFooter
        hide={totalPages <= 1}
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="Root Keys"
        loading={isInitialLoading}
        disabled={isNavigating}
      />
    </>
  );
}
