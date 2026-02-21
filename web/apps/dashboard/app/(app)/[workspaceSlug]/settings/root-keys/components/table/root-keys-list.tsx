"use client";
import { createRootKeyColumns, EmptyRootKeys, DataTable } from "@/components/data-table";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import type { UnkeyPermission } from "@unkey/rbac";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { useCallback, useMemo, useState } from "react";
import { RootKeyDialog } from "../dialog/root-key-dialog";

// Type guard function to check if a string is a valid UnkeyPermission
const isUnkeyPermission = (permissionName: string): permissionName is UnkeyPermission => {
  const result = unkeyPermissionValidation.safeParse(permissionName);
  return result.success;
};
import { renderRootKeySkeletonRow } from "@/components/data-table/components/skeletons/render-root-key-skeleton-row";
import { useRootKeysListQuery } from "@/components/data-table/hooks/rootkey/use-root-keys-list-query";
import { getRowClassName } from "@/components/data-table/utils/get-row-class";

export const RootKeysList = () => {
  const { rootKeys, isLoading, isLoadingMore, loadMore, totalCount, hasMore } =
    useRootKeysListQuery();
  const [selectedRootKey, setSelectedRootKey] = useState<RootKey | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<RootKey | null>(null);

  const handleEditKey = useCallback((rootKey: RootKey) => {
    setEditingKey(rootKey);
    setEditDialogOpen(true);
  }, []);

  // Memoize the selected root key ID to prevent unnecessary re-renders
  const selectedRootKeyId = selectedRootKey?.id;

  // Memoize the row click handler
  const handleRowClick = useCallback((rootKey: RootKey | null) => {
    if (rootKey) {
      setEditingKey(rootKey);
      setSelectedRootKey(rootKey);
      setEditDialogOpen(true);
    } else {
      setSelectedRootKey(null);
    }
  }, []);

  // Memoize the row className function
  const getRowClassNameMemoized = useCallback(
    (rootKey: RootKey) => getRowClassName(rootKey, selectedRootKey),
    [selectedRootKey],
  );

  // Memoize the loadMoreFooterProps to prevent unnecessary re-renders
  const loadMoreFooterProps = useMemo(
    () => ({
      hide: isLoading,
      buttonText: "Load more root keys",
      hasMore,
      countInfoText: (
        <div className="flex gap-2">
          <span>Showing</span>{" "}
          <span className="text-accent-12">{new Intl.NumberFormat().format(rootKeys.length)}</span>
          <span>of</span>
          {new Intl.NumberFormat().format(totalCount)}
          <span>root keys</span>
        </div>
      ),
    }),
    [isLoading, hasMore, rootKeys.length, totalCount],
  );

  // Memoize the emptyState to prevent unnecessary re-renders
  const emptyState = useMemo(
    () => (
      <EmptyRootKeys />
    ),
    [],
  );

  // Memoize the config to prevent unnecessary re-renders
  const config = useMemo(
    () => ({
      rowHeight: 40,
      layout: "grid" as const,
      rowBorders: true,
      containerPadding: "px-0",
      
    }),
    [],
  );

  // Memoize the renderSkeletonRow function to prevent unnecessary re-renders
  const renderSkeletonRow = useCallback(renderRootKeySkeletonRow, []);

  // Memoize the existingKey object to prevent unnecessary re-renders
  const existingKey = useMemo(() => {
    if (!editingKey) {
      return null;
    }

    // Guard against undefined permissions and use type guard function
    const permissions = editingKey.permissions ?? [];
    const validatedPermissions = permissions.map((p) => p.name).filter(isUnkeyPermission);

    return {
      id: editingKey.id,
      name: editingKey.name,
      permissions: validatedPermissions,
    };
  }, [editingKey]);

  const columns = useMemo(
    () => createRootKeyColumns({ selectedRootKeyId, onEditKey: handleEditKey }),
    [selectedRootKeyId, handleEditKey],
  );

  return (
    <>
      <DataTable
        data={rootKeys}
        columns={columns}
        getRowId={(rootKey) => rootKey.id}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        hasMore={hasMore}
        onRowClick={handleRowClick}
        selectedItem={selectedRootKey}
        rowClassName={getRowClassNameMemoized}
        loadMoreFooterProps={loadMoreFooterProps}
        emptyState={emptyState}
        config={config}
        renderSkeletonRow={renderSkeletonRow}
      />
      {editingKey && existingKey && (
        <RootKeyDialog
          title="Edit root key"
          subTitle="Update the name and permissions for this root key"
          isOpen={editDialogOpen}
          onOpenChange={(open) => {
            setEditDialogOpen(open);
            if (!open) {
              setEditingKey(null);
            }
          }}
          editMode={true}
          existingKey={existingKey}
        />
      )}
    </>
  );
};
