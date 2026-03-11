"use client";
import {
  createRootKeyColumns,
  getRowClassName,
  renderRootKeySkeletonRow,
  useRootKeysListPaginated,
} from "@/components/root-keys-table";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import type { UnkeyPermission } from "@unkey/rbac";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { DataTable, EmptyRootKeys, Loading, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import { RootKeyDialog } from "../dialog/root-key-dialog";

// Type guard function to check if a string is a valid UnkeyPermission
const isUnkeyPermission = (permissionName: string): permissionName is UnkeyPermission => {
  const result = unkeyPermissionValidation.safeParse(permissionName);
  return result.success;
};

const TABLE_CONFIG = {
  rowHeight: 40,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

export const RootKeysList = () => {
  const {
    rootKeys,
    isLoading,
    isInitialLoading,
    isFetching,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
    sorting,
    onSortingChange,
  } = useRootKeysListPaginated();
  const [selectedRootKey, setSelectedRootKey] = useState<RootKey | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<RootKey | null>(null);

  const handleEditKey = useCallback((rootKey: RootKey) => {
    setEditingKey(rootKey);
    setEditDialogOpen(true);
  }, []);

  const selectedRootKeyId = selectedRootKey?.id;

  const handleRowClick = useCallback((rootKey: RootKey | null) => {
    if (rootKey) {
      setEditingKey(rootKey);
      setSelectedRootKey(rootKey);
      setEditDialogOpen(true);
    } else {
      setSelectedRootKey(null);
    }
  }, []);

  const getRowClassNameMemoized = useCallback(
    (rootKey: RootKey) => getRowClassName(rootKey, selectedRootKey),
    [selectedRootKey],
  );

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

  const isNavigating = isFetching && !isInitialLoading;

  return (
    <>
      {isNavigating && (
        <div className="fixed inset-0 z-20 flex items-center justify-center pointer-events-none">
          <Loading size={32} className="text-accent-9" />
        </div>
      )}
      <DataTable
        data={rootKeys}
        columns={columns}
        getRowId={(rootKey) => rootKey.id}
        isLoading={isLoading || isFetching}
        onRowClick={handleRowClick}
        selectedItem={selectedRootKey}
        rowClassName={getRowClassNameMemoized}
        emptyState={<EmptyRootKeys />}
        config={TABLE_CONFIG}
        renderSkeletonRow={renderRootKeySkeletonRow}
        sorting={sorting}
        onSortingChange={onSortingChange}
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="root keys"
        loading={isInitialLoading}
        disabled={isNavigating}
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
