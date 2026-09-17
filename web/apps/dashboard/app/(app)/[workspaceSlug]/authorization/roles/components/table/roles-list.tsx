"use client";
import {
  createRolesColumns,
  renderRolesSkeletonRow,
  useRolesListPaginated,
} from "@/components/roles-table";
import { EditRole } from "@/components/roles-table/components/actions/components/edit-role";
import { SelectionControls } from "@/components/roles-table/components/selection-controls";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { IconBookBookmarkOutline18, IconShieldKeyOutline18 } from "@unkey/icons";
import {
  Button,
  DataTable,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PaginationFooter,
  getSelectableRowClassName,
} from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

export const RolesList = () => {
  const {
    roles,
    isInitialLoading,
    isNavigating,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  } = useRolesListPaginated();

  const [selectedRole, setSelectedRole] = useState<RoleBasic | null>(null);
  const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set());
  const [hoveredRoleName, setHoveredRoleName] = useState<string | null>(null);

  const toggleSelection = useCallback((roleId: string) => {
    setSelectedRoles((prevSelected) => {
      const newSelected = new Set(prevSelected);
      if (newSelected.has(roleId)) {
        newSelected.delete(roleId);
      } else {
        newSelected.add(roleId);
      }
      return newSelected;
    });
  }, []);

  const columns = useMemo(
    () =>
      createRolesColumns({
        selectedRoleId: selectedRole?.roleId,
        selectedRoles,
        hoveredRoleName,
        onToggleSelection: toggleSelection,
        onHoverRole: setHoveredRoleName,
      }),
    [selectedRole?.roleId, selectedRoles, hoveredRoleName, toggleSelection],
  );

  return (
    <>
      <DataTable
        data={roles}
        columns={columns}
        getRowId={(role) => role.roleId}
        isLoading={isInitialLoading}
        enableSorting={true}
        manualSorting={true}
        sorting={sorting}
        onSortingChange={onSortingChange}
        onRowClick={setSelectedRole}
        selectedItem={selectedRole}
        rowClassName={(role) => getSelectableRowClassName(role.roleId === selectedRole?.roleId)}
        renderSkeletonRow={renderRolesSkeletonRow}
        emptyState={
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconShieldKeyOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No Roles Found</EmptyStateTitle>
              <EmptyStateDescription>
                There are no roles configured yet. Create your first role to start managing
                permissions and access control.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/platform/apis/features/authorization/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="outline" size="md">
                  <IconBookBookmarkOutline18 />
                  Learn about Roles
                </Button>
              </a>
            </EmptyStateActions>
          </EmptyState>
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
        itemLabel="roles"
        hide={isInitialLoading}
        disabled={isNavigating}
        headerContent={
          <SelectionControls selectedRoles={selectedRoles} setSelectedRoles={setSelectedRoles} />
        }
      />
      {selectedRole && (
        <EditRole
          role={selectedRole}
          isOpen={!!selectedRole}
          onClose={() => setSelectedRole(null)}
        />
      )}
    </>
  );
};
