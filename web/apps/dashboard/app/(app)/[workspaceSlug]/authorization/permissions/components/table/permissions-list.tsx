"use client";
import {
  createPermissionsColumns,
  getRowClassName,
  renderPermissionsSkeletonRow,
  usePermissionsListPaginated,
} from "@/components/permissions-table";
import { EditPermission } from "@/components/permissions-table/components/actions/components/edit-permission";
import { SelectionControls } from "@/components/permissions-table/components/selection-controls";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

export const PermissionsList = () => {
  const { permissions, isLoading, page, pageSize, totalPages, totalCount, onPageChange } =
    usePermissionsListPaginated();

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
      <SelectionControls
        selectedPermissions={selectedPermissions}
        setSelectedPermissions={setSelectedPermissions}
      />
      <DataTable
        data={permissions}
        columns={columns}
        getRowId={(permission) => permission.permissionId}
        isLoading={isLoading}
        onRowClick={setSelectedPermission}
        selectedItem={selectedPermission}
        rowClassName={(permission) => getRowClassName(permission, selectedPermission)}
        renderSkeletonRow={renderPermissionsSkeletonRow}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No Permissions Found</Empty.Title>
              <Empty.Description className="text-left">
                There are no permissions configured yet. Create your first permission to start
                managing permissions and access control.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <a
                  href="https://www.unkey.com/docs/apis/features/authorization/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Learn about Permissions
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        }
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
