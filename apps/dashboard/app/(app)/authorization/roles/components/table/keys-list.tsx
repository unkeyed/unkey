"use client";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { Roles } from "@/lib/trpc/routers/authorization/roles";
import { BookBookmark, CircleInfoSparkle, Shield } from "@unkey/icons";
import { AnimatedLoadingSpinner, Button, Checkbox, Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import Link from "next/link";
import { useCallback, useMemo, useState } from "react";
import { useRolesListQuery } from "./hooks/use-roles-list-query";
import { getRowClassName } from "./utils/get-row-class";

export const RolesList = () => {
  const { roles, isLoading, isLoadingMore, loadMore, totalCount, hasMore } = useRolesListQuery();
  const [selectedRole, setSelectedRole] = useState<Roles | null>(null);
  const [navigatingRoleSlug, setNavigatingRoleSlug] = useState<string | null>(null);
  const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set());
  const [hoveredRoleSlug, setHoveredRoleSlug] = useState<string | null>(null);

  const handleLinkClick = useCallback((roleSlug: string) => {
    setNavigatingRoleSlug(roleSlug);
    setSelectedRole(null);
  }, []);

  const toggleSelection = useCallback((roleSlug: string) => {
    setSelectedRoles((prevSelected) => {
      const newSelected = new Set(prevSelected);
      if (newSelected.has(roleSlug)) {
        newSelected.delete(roleSlug);
      } else {
        newSelected.add(roleSlug);
      }
      return newSelected;
    });
  }, []);

  const columns: Column<Roles>[] = useMemo(
    () => [
      {
        key: "role",
        header: "Role",
        width: "25%",
        headerClassName: "pl-[18px]",
        render: (role) => {
          const isNavigating = role.slug === navigatingRoleSlug;
          const isSelected = selectedRoles.has(role.slug);
          const isHovered = hoveredRoleSlug === role.slug;

          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center cursor-pointer",
                "bg-brand-3",
                isSelected && "bg-brand-5",
              )}
              onMouseEnter={() => setHoveredRoleSlug(role.slug)}
              onMouseLeave={() => setHoveredRoleSlug(null)}
            >
              {isNavigating ? (
                <div className="text-brand-11">
                  <AnimatedLoadingSpinner />
                </div>
              ) : (
                <>
                  {!isSelected && !isHovered && (
                    <Shield size="md-regular" className="text-brand-11" />
                  )}

                  {(isSelected || isHovered) && (
                    <Checkbox
                      checked={isSelected}
                      className="size-4 [&_svg]:size-3"
                      onCheckedChange={() => toggleSelection(role.slug)}
                    />
                  )}
                </>
              )}
            </div>
          );

          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-4 items-center">
                {iconContainer}
                <div className="flex flex-col gap-1 text-xs">
                  <Link
                    title={`View details for ${role.slug}`}
                    className="font-mono group-hover:underline decoration-dotted"
                    href={`/roles/${role.slug}`}
                    aria-disabled={isNavigating}
                    onClick={() => {
                      handleLinkClick(role.slug);
                    }}
                  >
                    <div className="font-mono font-medium truncate text-brand-12">{role.slug}</div>
                  </Link>
                  {role.name && (
                    <span
                      className="font-sans text-accent-9 truncate max-w-[180px]"
                      title={role.name}
                    >
                      {role.name}
                    </span>
                  )}
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "description",
        header: "Description",
        width: "25%",
        render: (role) => (
          <div className="text-xs text-accent-11 max-w-[200px] truncate" title={role.description}>
            {role.description || <span className="text-accent-9 italic">No description</span>}
          </div>
        ),
      },
      {
        key: "assignedKeys",
        header: "Assigned Keys",
        width: "25%",
        render: (role) => (
          <AssignedItemsCell
            items={role.assignedKeys.items}
            totalCount={role.assignedKeys.totalCount}
            type="keys"
          />
        ),
      },
      {
        key: "permissions",
        header: "Permissions",
        width: "25%",
        render: (role) => (
          <AssignedItemsCell
            items={role.permissions.items}
            totalCount={role.permissions.totalCount}
            type="permissions"
          />
        ),
      },
    ],
    [navigatingRoleSlug, handleLinkClick, selectedRoles, toggleSelection, hoveredRoleSlug],
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
        keyExtractor={(role) => role.slug}
        rowClassName={(role) => getRowClassName(role, selectedRole)}
        loadMoreFooterProps={{
          hide: isLoading,
          buttonText: "Load more roles",
          hasMore,
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span> <span className="text-accent-12">{roles.length}</span>
              <span>of</span>
              {totalCount}
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
          columns.map((column, idx) => (
            <td
              key={column.key}
              className={cn(
                "text-xs align-middle whitespace-nowrap pr-4",
                idx === 0 ? "pl-[18px]" : "",
                column.key === "role" ? "py-[6px]" : "py-1",
              )}
              style={{ height: `${rowHeight}px` }}
            >
              {column.key === "role" && <RoleColumnSkeleton />}
              {column.key === "description" && <DescriptionColumnSkeleton />}
              {(column.key === "assignedKeys" || column.key === "permissions") && (
                <AssignedItemsColumnSkeleton />
              )}
              {!["role", "description", "assignedKeys", "permissions"].includes(column.key) && (
                <div className="h-4 w-full bg-grayA-3 rounded animate-pulse" />
              )}
            </td>
          ))
        }
      />
    </>
  );
};

const AssignedItemsCell = ({
  items,
  totalCount,
  type,
}: {
  items: string[];
  totalCount?: number;
  type: "keys" | "permissions";
}) => {
  const hasMore = totalCount && totalCount > items.length;
  const icon =
    type === "keys" ? <Shield className="size-3" /> : <CircleInfoSparkle className="size-3" />;

  if (items.length === 0) {
    return (
      <div className="flex items-center gap-2 text-xs text-accent-9">
        {icon}
        <span>None assigned</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2 text-xs">
        {icon}
        <span className="text-accent-12 font-medium">
          {totalCount ? totalCount : items.length} {type}
        </span>
      </div>
      <div className="flex flex-wrap gap-1">
        {items.map((item, idx) => (
          <Badge
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            key={idx}
            variant="secondary"
            className="text-xs py-0.5 px-1.5"
          >
            {item}
          </Badge>
        ))}
        {hasMore && (
          <Badge variant="secondary" className="text-xs py-0.5 px-1.5">
            +{(totalCount || 0) - items.length} more
          </Badge>
        )}
      </div>
    </div>
  );
};

const RoleColumnSkeleton = () => (
  <div className="flex items-center gap-4">
    <div className="size-5 bg-grayA-3 rounded animate-pulse" />
    <div className="flex flex-col gap-2">
      <div className="h-3 w-20 bg-grayA-3 rounded animate-pulse" />
      <div className="h-2 w-16 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);

const DescriptionColumnSkeleton = () => (
  <div className="h-3 w-32 bg-grayA-3 rounded animate-pulse" />
);

const AssignedItemsColumnSkeleton = () => (
  <div className="flex flex-col gap-2">
    <div className="flex items-center gap-2">
      <div className="size-3 bg-grayA-3 rounded animate-pulse" />
      <div className="h-3 w-12 bg-grayA-3 rounded animate-pulse" />
    </div>
    <div className="flex gap-1">
      <div className="h-5 w-16 bg-grayA-3 rounded animate-pulse" />
      <div className="h-5 w-12 bg-grayA-3 rounded animate-pulse" />
    </div>
  </div>
);
