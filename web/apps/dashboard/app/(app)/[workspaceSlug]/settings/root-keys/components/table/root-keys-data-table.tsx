"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { IconKey2Outline18 } from "@unkey/icons";
import {
  EmptyRootKeys,
  PaginationFooter,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
  TimestampInfo,
} from "@unkey/ui";
import { cn } from "cn";
import dynamic from "next/dynamic";
import { PermissionsCell } from "./permissions-cell";

const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3", "skeleton-4", "skeleton-5"];

const RootKeysTableActions = dynamic(
  () => import("./root-keys-table-action.popover").then((module) => module.RootKeysTableActions),
  { loading: () => <Skeleton className="size-7 rounded-md" />, ssr: false },
);

type RootKeysDataTableProps = {
  selectedKeyId: string | null;
  onEditKey: (rootKey: RootKey) => void;
  list: {
    rootKeys: RootKey[];
    isInitialLoading: boolean;
    isNavigating: boolean;
    totalCount: number;
    onPageChange: (page: number) => void;
    page: number;
    pageSize: number;
    totalPages: number;
  };
  transport: "legacy" | "v2";
};

export function RootKeysDataTable({
  selectedKeyId,
  onEditKey,
  list,
  transport,
}: RootKeysDataTableProps) {
  const {
    rootKeys,
    isInitialLoading,
    isNavigating,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
  } = list;

  return (
    <>
      {isInitialLoading ? (
        <RootKeysTableSkeleton />
      ) : rootKeys.length === 0 ? (
        <ResourceListContent>
          <div className="flex min-h-[320px] items-center justify-center px-4 py-16">
            <EmptyRootKeys />
          </div>
        </ResourceListContent>
      ) : (
        <ResourceListContent>
          <div className="overflow-x-auto">
            <div className="min-w-[640px]">
              <RootKeysTableHeader />
              <ResourceListBody aria-label="Root Keys">
                {rootKeys.map((rootKey) => (
                  <RootKeyRow
                    key={rootKey.id}
                    rootKey={rootKey}
                    selected={rootKey.id === selectedKeyId}
                    transport={transport}
                    onEditKey={onEditKey}
                  />
                ))}
              </ResourceListBody>
            </div>
          </div>
        </ResourceListContent>
      )}
      <PaginationFooter
        hide={totalPages <= 1}
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="Root Keys"
        loading={isInitialLoading}
        disabled={isNavigating}
      />
    </>
  );
}

function RootKeysTableHeader() {
  return (
    <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.9fr)_32px] items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
      <span>Name</span>
      <span>Permissions</span>
      <span>Created</span>
      <span />
    </div>
  );
}

function RootKeyRow({
  rootKey,
  selected,
  transport,
  onEditKey,
}: {
  rootKey: RootKey;
  selected: boolean;
  transport: "legacy" | "v2";
  onEditKey: (rootKey: RootKey) => void;
}) {
  const name = rootKey.name ?? "Unnamed Root Key";

  return (
    <ResourceListItem
      className={cn(
        "grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.9fr)_32px] items-center gap-4 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2",
        selected && "bg-grayA-2",
      )}
    >
      <button
        type="button"
        className="absolute inset-0 z-0 cursor-pointer rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
        aria-label={`Edit ${name}`}
        onClick={() => onEditKey(rootKey)}
      />
      <span className="flex min-w-0 items-center gap-2.5">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
          <IconKey2Outline18 className="text-gray-12" />
        </span>
        <span
          className={cn(
            "truncate text-sm font-medium",
            !rootKey.name && "font-normal italic text-gray-9",
          )}
        >
          {name}
        </span>
      </span>
      <span className="min-w-0">
        <PermissionsCell permissions={rootKey.permissions} />
      </span>
      <TimestampInfo
        value={rootKey.createdAt}
        displayType="relative"
        side="top"
        align="center"
        className="relative z-10 min-w-0 justify-self-start truncate text-xs text-gray-9"
      />
      <span className="relative z-10 flex justify-end">
        <RootKeysTableActions rootKey={rootKey} onEditKey={onEditKey} transport={transport} />
      </span>
    </ResourceListItem>
  );
}

function RootKeysTableSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading Root Keys...</output>
      <div className="overflow-x-auto">
        <div className="min-w-[640px]">
          <RootKeysTableHeader />
          <ResourceListBody aria-hidden="true">
            {SKELETON_KEYS.map((key) => (
              <ResourceListItem
                key={key}
                className="grid h-12 grid-cols-[minmax(0,1.2fr)_minmax(0,1.5fr)_minmax(0,0.9fr)_32px] items-center gap-4 px-4"
              >
                <span className="flex items-center gap-2.5">
                  <Skeleton className="size-6 shrink-0 rounded-md" />
                  <Skeleton className="h-3 w-28" />
                </span>
                <Skeleton className="h-3 w-36" />
                <Skeleton className="h-3 w-24" />
                <Skeleton className="size-7 rounded-md" />
              </ResourceListItem>
            ))}
          </ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}
