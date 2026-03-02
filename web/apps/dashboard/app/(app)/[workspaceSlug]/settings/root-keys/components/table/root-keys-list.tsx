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
import { DataTable, EmptyRootKeys, PaginationFooter } from "@unkey/ui";
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
    isFetching,
    isPending,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
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

  return (
    <>
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
        enableSorting={true}
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="root keys"
        hide={isPending}
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
