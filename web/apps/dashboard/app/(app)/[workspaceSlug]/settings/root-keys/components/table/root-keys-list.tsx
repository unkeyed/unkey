"use client";
import { HiddenValueCell } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/hidden-value";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import { BookBookmark, Dots, Key2 } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { Empty, InfoTooltip, TimestampInfo, buttonVariants } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useCallback, useMemo, useState } from "react";
import { RootKeyDialog } from "../root-key/root-key-dialog";

// Type guard function to check if a string is a valid UnkeyPermission
const isUnkeyPermission = (permissionName: string): permissionName is UnkeyPermission => {
  const result = unkeyPermissionValidation.safeParse(permissionName);
  return result.success;
};
import { AssignedItemsCell } from "./components/assigned-items-cell";
import { LastUpdated } from "./components/last-updated";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  KeyColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
  RootKeyColumnSkeleton,
} from "./components/skeletons";
import { useRootKeysListQuery } from "./hooks/use-root-keys-list-query";
import { getRowClassName } from "./utils/get-row-class";

const RootKeysTableActions = dynamic(
  () =>
    import("./components/actions/root-keys-table-action.popover.constants").then(
      (mod) => mod.RootKeysTableActions,
    ),
  {
    loading: () => (
      <button
        type="button"
        className={cn(
          "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded-sm m-0 items-center flex justify-center",
          "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
        )}
      >
        <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
      </button>
    ),
  },
);

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
  const handleRowClick = useCallback((rootKey: RootKey) => {
    setEditingKey(rootKey);
    setSelectedRootKey(rootKey);
    setEditDialogOpen(true);
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
      <div className="w-full flex justify-center items-center h-full">
        <Empty className="w-[400px] flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>No Root Keys Found</Empty.Title>
          <Empty.Description className="text-left">
            There are no root keys configured yet. Create your first root key to start managing
            permissions and access control.
          </Empty.Description>
          <Empty.Actions className="mt-4 justify-start">
            <a
              href="https://www.unkey.com/docs/security/root-keys"
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "outline" })}
            >
              <span className="flex items-center gap-2">
                <BookBookmark />
                Learn about Root Keys
              </span>
            </a>
          </Empty.Actions>
        </Empty>
      </div>
    ),
    [],
  );

  // Memoize the config to prevent unnecessary re-renders
  const config = useMemo(
    () => ({
      rowHeight: 52,
      layoutMode: "grid" as const,
      rowBorders: true,
      containerPadding: "px-0",
    }),
    [],
  );

  // Memoize the keyExtractor to prevent unnecessary re-renders
  const keyExtractor = useCallback((rootKey: RootKey) => rootKey.id, []);

  // Memoize the renderSkeletonRow function to prevent unnecessary re-renders
  const renderSkeletonRow = useCallback(
    ({
      columns,
      rowHeight,
    }: {
      columns: Column<RootKey>[];
      rowHeight: number;
    }) =>
      columns.map((column) => (
        <td
          key={column.key}
          className={cn(
            "text-xs align-middle whitespace-nowrap",
            column.key === "root_key" ? "py-[6px]" : "py-1",
            column.cellClassName,
          )}
          style={{ height: `${rowHeight}px` }}
        >
          {column.key === "root_key" && <RootKeyColumnSkeleton />}
          {column.key === "key" && <KeyColumnSkeleton />}
          {column.key === "created_at" && <CreatedAtColumnSkeleton />}
          {column.key === "permissions" && <PermissionsColumnSkeleton />}
          {column.key === "last_updated" && <LastUpdatedColumnSkeleton />}
          {column.key === "action" && <ActionColumnSkeleton />}
        </td>
      )),
    [],
  );

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

  const columns: Column<RootKey>[] = useMemo(
    () => [
      {
        key: "root_key",
        header: "Name",
        width: "15%",
        headerClassName: "pl-[18px]",
        render: (rootKey) => {
          const isSelected = rootKey.id === selectedRootKeyId;
          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded-sm flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5",
              )}
            >
              <Key2 iconSize="sm-regular" className="text-gray-12" />
            </div>
          );
          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-3 items-center w-full">
                {iconContainer}
                <div className="w-[150px]">
                  <div
                    className={cn(
                      "font-medium truncate leading-4 text-[13px]",
                      rootKey.name ? "text-accent-12" : "text-gray-9 italic font-normal",
                    )}
                  >
                    {rootKey.name ?? "Unnamed Root Key"}
                  </div>
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "key",
        header: "Key",
        width: "15%",
        render: (rootKey) => (
          <InfoTooltip
            content={
              <p>
                This is the first part of the key to visually match it. We don't store the full key
                for security reasons.
              </p>
            }
          >
            <HiddenValueCell
              value={rootKey.start}
              title="Key"
              selected={selectedRootKeyId === rootKey.id}
            />
          </InfoTooltip>
        ),
      },
      {
        key: "permissions",
        header: "Permissions",
        width: "15%",
        render: (rootKey) => (
          <AssignedItemsCell
            isSelected={rootKey.id === selectedRootKeyId}
            permissionSummary={rootKey.permissionSummary}
          />
        ),
      },
      {
        key: "created_at",
        header: "Created At",
        width: "20%",
        render: (rootKey) => {
          return (
            <TimestampInfo
              value={rootKey.createdAt}
              className={cn("font-mono group-hover:underline decoration-dotted")}
            />
          );
        },
      },
      {
        key: "last_updated",
        header: "Last Updated",
        width: "20%",
        render: (rootKey) => {
          return (
            <LastUpdated
              lastUpdated={rootKey.lastUpdatedAt ?? 0}
              isSelected={rootKey.id === selectedRootKeyId}
            />
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "auto",
        render: (rootKey) => {
          return <RootKeysTableActions rootKey={rootKey} onEditKey={handleEditKey} />;
        },
      },
    ],
    [selectedRootKeyId, handleEditKey],
  );

  return (
    <>
      <VirtualTable
        data={rootKeys}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns}
        onRowClick={handleRowClick}
        selectedItem={selectedRootKey}
        keyExtractor={keyExtractor}
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
