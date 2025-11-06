"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { BookBookmark, Tag } from "@unkey/icons";
import { Button, Checkbox, Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { EditRole } from "./components/actions/components/edit-role";
import { RolesTableActions } from "./components/actions/keys-table-action.popover.constants";
import { AssignedItemsCell } from "./components/assigned-items-cell";
import { LastUpdated } from "./components/last-updated";
import { SelectionControls } from "./components/selection-controls";
import {
  ActionColumnSkeleton,
  AssignedKeysColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
  RoleColumnSkeleton,
  SlugColumnSkeleton,
} from "./components/skeletons";
import { useRolesListQuery } from "./hooks/use-roles-list-query";
import { getRowClassName } from "./utils/get-row-class";

export const RolesList = () => {
  const { roles, isLoading, isLoadingMore, loadMore, totalCount, hasMore } = useRolesListQuery();
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

  const columns: Column<RoleBasic>[] = useMemo(
    () => [
      {
        key: "role",
        header: "Role",
        width: "20%",
        headerClassName: "pl-[18px]",
        render: (role) => {
          const isSelected = selectedRoles.has(role.roleId);
          const isHovered = hoveredRoleName === role.name;

          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5",
              )}
              onMouseEnter={() => setHoveredRoleName(role.name)}
              onMouseLeave={() => setHoveredRoleName(null)}
            >
              {!isSelected && !isHovered && (
                <Tag
                  iconSize="sm-regular"
                  className="text-gray-12 cursor-pointer w-[12px] h-[12px]"
                />
              )}
              {(isSelected || isHovered) && (
                <Checkbox
                  checked={isSelected}
                  className="size-4 [&_svg]:size-3"
                  onClick={(e) => e.stopPropagation()}
                  onCheckedChange={() => toggleSelection(role.roleId)}
                />
              )}
            </div>
          );

          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-4 items-center">
                {iconContainer}
                <div className="flex flex-col gap-1 text-xs">
                  <div className="font-medium truncate text-accent-12 leading-4 text-[13px] max-w-[120px]">
                    {role.name}
                  </div>
                  {role.description ? (
                    <span
                      className="font-sans text-accent-9 truncate max-w-[180px] text-xs"
                      title={role.description}
                    >
                      {role.description}
                    </span>
                  ) : (
                    <span className="font-sans text-accent-9 truncate max-w-[180px] text-xs italic">
                      No description
                    </span>
                  )}
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "assignedKeys",
        header: "Assigned Keys",
        width: "20%",
        render: (role) => (
          <AssignedItemsCell
            roleId={role.roleId}
            isSelected={role.roleId === selectedRole?.roleId}
            kind="keys"
          />
        ),
      },
      {
        key: "permissions",
        header: "Assigned Permissions",
        width: "20%",
        render: (role) => (
          <AssignedItemsCell
            roleId={role.roleId}
            isSelected={role.roleId === selectedRole?.roleId}
            kind="permissions"
          />
        ),
      },
      {
        key: "last_updated",
        header: "Last Updated",
        width: "12%",
        render: (role) => {
          return (
            <LastUpdated
              lastUpdated={role.lastUpdated}
              isSelected={role.roleId === selectedRole?.roleId}
            />
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "15%",
        render: (role) => {
          return <RolesTableActions role={role} />;
        },
      },
    ],
    [selectedRoles, toggleSelection, hoveredRoleName, selectedRole?.roleId],
  );

  return (
    <>
      <VirtualTable
        data={roles}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns}
        onRowClick={setSelectedRole}
        selectedItem={selectedRole}
        keyExtractor={(role) => role.roleId}
        rowClassName={(role) => getRowClassName(role, selectedRole)}
        loadMoreFooterProps={{
          hide: isLoading,
          buttonText: "Load more roles",
          hasMore,
          headerContent: (
            <SelectionControls selectedRoles={selectedRoles} setSelectedRoles={setSelectedRoles} />
          ),
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span>{" "}
              <span className="text-accent-12">{new Intl.NumberFormat().format(roles.length)}</span>
              <span>of</span>
              {new Intl.NumberFormat().format(totalCount)}
              <span>roles</span>
            </div>
          ),
        }}
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
                  href="https://www.unkey.com/docs/security/roles"
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
          layoutMode: "grid",
          rowBorders: true,
          containerPadding: "px-0",
        }}
        renderSkeletonRow={({ columns, rowHeight }) =>
          columns.map((column) => (
            <td
              key={column.key}
              className={cn(
                "text-xs align-middle whitespace-nowrap",
                column.key === "role" ? "py-[6px]" : "py-1",
              )}
              style={{ height: `${rowHeight}px` }}
            >
              {column.key === "role" && <RoleColumnSkeleton />}
              {column.key === "slug" && <SlugColumnSkeleton />}
              {column.key === "assignedKeys" && <AssignedKeysColumnSkeleton />}
              {column.key === "permissions" && <PermissionsColumnSkeleton />}
              {column.key === "last_updated" && <LastUpdatedColumnSkeleton />}
              {column.key === "action" && <ActionColumnSkeleton />}
            </td>
          ))
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
