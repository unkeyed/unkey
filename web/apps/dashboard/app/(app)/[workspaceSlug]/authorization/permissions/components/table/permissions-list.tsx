"use client";
import {
  EmptyPermissions,
  createPermissionsColumns,
  getRowClassName,
  renderPermissionsSkeletonRow,
  usePermissionsListPaginated,
} from "@/components/permissions-table";
import { EditPermission } from "@/components/permissions-table/components/actions/components/edit-permission";
import { SelectionControls } from "@/components/permissions-table/components/selection-controls";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { DataTable, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

export const PermissionsList = () => {
  const {
    permissions,
    isLoading,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  } = usePermissionsListPaginated();

  const [selectedPermission, setSelectedPermission] = useState<Permission | null>(null);
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set());
  const [hoveredPermissionName, setHoveredPermissionName] = useState<string | null>(null);

  const toggleSelection = useCallback((permissionId: string) => {
    setSelectedPermissions((prevSelected) => {
      const newSelected = new Set(prevSelected);
      if (newSelected.has(permissionId)) {
        newSelected.delete(permissionId);
      } else {
        newSelected.add(permissionId);
      }
      return newSelected;
    });
  }, []);

  const columns = useMemo(
    () =>
      createPermissionsColumns({
        selectedPermissionId: selectedPermission?.permissionId,
        selectedPermissions,
        hoveredPermissionName,
        onToggleSelection: toggleSelection,
        onHoverPermission: setHoveredPermissionName,
      }),
    [selectedPermission?.permissionId, selectedPermissions, hoveredPermissionName, toggleSelection],
  );

  return (
    <>
      <DataTable
        data={permissions}
        columns={columns}
        getRowId={(permission) => permission.permissionId}
        isLoading={isLoading}
        onRowClick={setSelectedPermission}
        selectedItem={selectedPermission}
        rowClassName={(permission) => getRowClassName(permission, selectedPermission)}
        renderSkeletonRow={renderPermissionsSkeletonRow}
        emptyState={<EmptyPermissions />}
        enableSorting={true}
        manualSorting={true}
        sorting={sorting}
        onSortingChange={onSortingChange}
        config={{
          rowHeight: 52,
          layout: "grid",
          rowBorders: true,
          containerPadding: "px-0",
        }}
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        headerContent={
          <SelectionControls
            selectedPermissions={selectedPermissions}
            setSelectedPermissions={setSelectedPermissions}
          />
        }
      />
      {selectedPermission && (
        <EditPermission
          permission={selectedPermission}
          isOpen={!!selectedPermission}
          onClose={() => setSelectedPermission(null)}
        />
      )}
    </>
  );
};
