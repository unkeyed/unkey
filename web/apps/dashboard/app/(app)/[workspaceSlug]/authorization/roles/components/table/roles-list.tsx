"use client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, PaginationFooter } from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import {
  createRolesColumns,
  getRowClassName,
  renderRolesSkeletonRow,
  useRolesListPaginated,
} from "@/components/roles-table";
import { EditRole } from "./components/actions/components/edit-role";
import { SelectionControls } from "./components/selection-controls";

export const RolesList = () => {
  const {
    roles,
    isLoading,
    isFetching,
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
        isLoading={isLoading}
        enableSorting={true}
        manualSorting={true}
        sorting={sorting}
        onSortingChange={onSortingChange}
        onRowClick={setSelectedRole}
        rowClassName={(role) => getRowClassName(role, selectedRole)}
        renderSkeletonRow={renderRolesSkeletonRow}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No Roles Found</Empty.Title>
              <Empty.Description className="text-left">
                There are no roles configured yet. Create your first role to start managing
                permissions and access control.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <a
                  href="https://www.unkey.com/docs/apis/features/authorization/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Learn about Roles
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
        itemLabel="roles"
        hide={isLoading}
        disabled={isFetching}
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
