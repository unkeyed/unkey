"use client";
import { HiddenValueCell } from "@/app/(app)/[workspaceId]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/hidden-value";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { BookBookmark, Dots, Key2 } from "@unkey/icons";
import { Button, Empty, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import { useMemo, useState } from "react";
import { AssignedItemsCell } from "./components/assigned-items-cell";
import { CriticalPermissionIndicator } from "./components/critical-perm-warning";
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
          "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
          "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
        )}
      >
        <Dots className="group-hover:text-gray-12 text-gray-11" size="sm-regular" />
      </button>
    ),
  },
);

export const RootKeysList = ({ workspaceId }: { workspaceId: string }) => {
  const { rootKeys, isLoading, isLoadingMore, loadMore, totalCount, hasMore } =
    useRootKeysListQuery();
  const [selectedRootKey, setSelectedRootKey] = useState<RootKey | null>(null);

  const columns: Column<RootKey>[] = useMemo(
    () => [
      {
        key: "root_key",
        header: "Name",
        width: "15%",
        headerClassName: "pl-[18px]",
        render: (rootKey) => {
          const isSelected = rootKey.id === selectedRootKey?.id;
          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5",
              )}
            >
              <Key2 size="sm-regular" className="text-gray-12" />
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
                <CriticalPermissionIndicator rootKey={rootKey} isSelected={isSelected} />
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
              selected={selectedRootKey?.id === rootKey.id}
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
            isSelected={rootKey.id === selectedRootKey?.id}
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
              isSelected={rootKey.id === selectedRootKey?.id}
            />
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "auto",
        render: (rootKey) => {
          return <RootKeysTableActions rootKey={rootKey} workspaceId={workspaceId} />;
        },
      },
    ],
    [selectedRootKey?.id, workspaceId],
  );

  return (
    <VirtualTable
      data={rootKeys}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedRootKey}
      selectedItem={selectedRootKey}
      keyExtractor={(rootKey) => rootKey.id}
      rowClassName={(rootKey) => getRowClassName(rootKey, selectedRootKey)}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more root keys",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span> <span className="text-accent-12">{rootKeys.length}</span>
            <span>of</span>
            {totalCount}
            <span>root keys</span>
          </div>
        ),
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Root Keys Found</Empty.Title>
            <Empty.Description className="text-left">
              There are no root keys configured yet. Create your first role to start managing
              permissions and access control.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/security/root-keys"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Learn about Root Keys
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
              column.key === "root_key" ? "py-[6px]" : "py-1",
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
        ))
      }
    />
  );
};
