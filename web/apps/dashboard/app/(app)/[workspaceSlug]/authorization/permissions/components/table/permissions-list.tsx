"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { BookBookmark, Page2 } from "@unkey/icons";
import { Button, Checkbox, Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { EditPermission } from "./components/actions/components/edit-permission";
import { PermissionsTableActions } from "./components/actions/keys-table-action.popover.constants";
import { AssignedItemsCell } from "./components/assigned-items-cell";
import { LastUpdated } from "./components/last-updated";
import { SelectionControls } from "./components/selection-controls";
import {
  ActionColumnSkeleton,
  AssignedKeysColumnSkeleton,
  AssignedToKeysColumnSkeleton,
  LastUpdatedColumnSkeleton,
  RoleColumnSkeleton,
  SlugColumnSkeleton,
} from "./components/skeletons";
import { usePermissionsListQuery } from "./hooks/use-permissions-list-query";
import { getRowClassName } from "./utils/get-row-class";

export const PermissionsList = () => {
  const { permissions, isLoading, isLoadingMore, loadMore, totalCount, hasMore } =
    usePermissionsListQuery();
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

  const columns: Column<Permission>[] = useMemo(
    () => [
      {
        key: "permission",
        header: "Permission",
        width: "20%",
        headerClassName: "pl-[18px]",
        render: (permission) => {
          const isSelected = selectedPermissions.has(permission.permissionId);
          const isHovered = hoveredPermissionName === permission.name;

          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5",
              )}
              onMouseEnter={() => setHoveredPermissionName(permission.name)}
              onMouseLeave={() => setHoveredPermissionName(null)}
            >
              {!isSelected && !isHovered && (
                <Page2 iconSize="sm-regular" className="text-gray-12 cursor-pointer" />
              )}
              {(isSelected || isHovered) && (
                <Checkbox
                  checked={isSelected}
                  className="size-4 [&_svg]:size-3"
                  onClick={(e) => e.stopPropagation()}
                  onCheckedChange={() => toggleSelection(permission.permissionId)}
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
                    {permission.name}
                  </div>
                  {permission.description ? (
                    <span
                      className="font-sans text-accent-9 truncate max-w-[180px] text-xs"
                      title={permission.description}
                    >
                      {permission.description}
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
        key: "slug",
        header: "Slug",
        width: "20%",
        render: (permission) => (
          <AssignedItemsCell
            kind="slug"
            value={permission.slug}
            isSelected={permission.permissionId === selectedPermission?.permissionId}
          />
        ),
      },
      {
        key: "used_in_roles",
        header: "Used in Roles",
        width: "20%",
        render: (permission) => (
          <AssignedItemsCell
            kind="roles"
            totalCount={permission.totalConnectedRoles}
            isSelected={permission.permissionId === selectedPermission?.permissionId}
          />
        ),
      },
      {
        key: "assigned_to_keys",
        header: "Assigned to Keys",
        width: "20%",
        render: (permission) => (
          <AssignedItemsCell
            kind="keys"
            totalCount={permission.totalConnectedKeys}
            isSelected={permission.permissionId === selectedPermission?.permissionId}
          />
        ),
      },
      {
        key: "last_updated",
        header: "Last Updated",
        width: "12%",
        render: (permission) => {
          return (
            <LastUpdated
              lastUpdated={permission.lastUpdated}
              isSelected={permission.permissionId === selectedPermission?.permissionId}
            />
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "15%",
        render: (permission) => {
          return <PermissionsTableActions permission={permission} />;
        },
      },
    ],
    [selectedPermissions, toggleSelection, hoveredPermissionName, selectedPermission?.permissionId],
  );

  return (
    <>
      <VirtualTable
        data={permissions}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns}
        onRowClick={setSelectedPermission}
        selectedItem={selectedPermission}
        keyExtractor={(permission) => permission.permissionId}
        rowClassName={(permission) => getRowClassName(permission, selectedPermission)}
        loadMoreFooterProps={{
          hide: isLoading,
          buttonText: "Load more permissions",
          hasMore,
          headerContent: (
            <SelectionControls
              selectedPermissions={selectedPermissions}
              setSelectedPermissions={setSelectedPermissions}
            />
          ),
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span>{" "}
              <span className="text-accent-12">
                {new Intl.NumberFormat().format(permissions.length)}
              </span>
              <span>of</span>
              {new Intl.NumberFormat().format(totalCount)}
              <span>permissions</span>
            </div>
          ),
        }}
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
                column.key === "permission" ? "py-[6px]" : "py-1",
                column.cellClassName,
              )}
              style={{ height: `${rowHeight}px` }}
            >
              {column.key === "permission" && <RoleColumnSkeleton />}
              {column.key === "slug" && <SlugColumnSkeleton />}
              {column.key === "used_in_roles" && <AssignedKeysColumnSkeleton />}
              {column.key === "assigned_to_keys" && <AssignedToKeysColumnSkeleton />}
              {column.key === "last_updated" && <LastUpdatedColumnSkeleton />}
              {column.key === "action" && <ActionColumnSkeleton />}
            </td>
          ))
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
