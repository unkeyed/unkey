"use client";

import { LastUsedCell } from "@/components/api-keys-table/components/last-used";
import { IconKey2Outline12, IconPlusOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
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
import type { RootKey } from "./root-keys-v2";

const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3", "skeleton-4", "skeleton-5"];

const RootKeysTableActions = dynamic(
  () => import("./root-keys-table-action.popover").then((module) => module.RootKeysTableActions),
  { loading: () => <Skeleton className="size-7 rounded-md" />, ssr: false },
);

type RootKeysDataTableProps = {
  selectedKeyId: string | null;
  onEditKey: (rootKey: RootKey) => void;
  onCreate: () => void;
  list: {
    rootKeys: RootKey[];
    isInitialLoading: boolean;
    isNavigating: boolean;
    totalCount: number;
    onPageChange: (page: number) => void;
    page: number;
    pageSize: number;
    totalPages: number;
    isError: boolean;
    retry: () => void;
    search: string;
    clearFilters: () => void;
  };
};

export function RootKeysDataTable({
  selectedKeyId,
  onEditKey,
  onCreate,
  list,
}: RootKeysDataTableProps) {
  const { isInitialLoading, isNavigating, totalCount, onPageChange, page, pageSize, totalPages } =
    list;

  return (
    <>
      <RootKeysTableBody
        selectedKeyId={selectedKeyId}
        onEditKey={onEditKey}
        onCreate={onCreate}
        list={list}
      />
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

function RootKeysTableBody({ selectedKeyId, onEditKey, onCreate, list }: RootKeysDataTableProps) {
  const { rootKeys, isInitialLoading, isError, retry, search, clearFilters } = list;

  if (isInitialLoading) {
    return <RootKeysTableSkeleton />;
  }

  if (isError && rootKeys.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center gap-3 px-4 py-16">
          <span role="alert" className="text-sm text-error-11">
            Unable to load Root Keys. Check your connection and try again.
          </span>
          <Button size="sm" variant="outline" className="rounded-md px-3" onClick={retry}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  if (rootKeys.length === 0 && search !== "") {
    return (
      <EmptyState>
        <EmptyStateHeader>
          <EmptyStateTitle className="max-w-md break-words">
            No Root Keys match “{search}”
          </EmptyStateTitle>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button size="sm" variant="outline" className="rounded-md px-3" onClick={clearFilters}>
            Clear search
          </Button>
        </EmptyStateActions>
      </EmptyState>
    );
  }

  if (rootKeys.length === 0) {
    return (
      <EmptyState>
        <EmptyStateHeader>
          <EmptyStateTitle>No Root Keys yet</EmptyStateTitle>
          <EmptyStateDescription>
            Root Keys give your services access to the Unkey API.
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button size="sm" variant="primary" className="rounded-md px-3" onClick={onCreate}>
            <IconPlusOutline18 />
            New Root Key
          </Button>
        </EmptyStateActions>
      </EmptyState>
    );
  }

  return (
    <ResourceListContent>
      <div className="overflow-x-auto">
        <div className="min-w-[960px]">
          <RootKeysTableHeader />
          <ResourceListBody aria-label="Root Keys">
            {rootKeys.map((rootKey) => (
              <RootKeyRow
                key={rootKey.id}
                rootKey={rootKey}
                selected={rootKey.id === selectedKeyId}
                onEditKey={onEditKey}
              />
            ))}
          </ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}

function RootKeysTableHeader() {
  return (
    <div className="grid grid-cols-[minmax(0,1.1fr)_160px_minmax(0,1.3fr)_minmax(0,0.8fr)_minmax(0,0.8fr)_32px] items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
      <span>Name</span>
      <span>Value</span>
      <span>Permissions</span>
      <span>Last used</span>
      <span>Created</span>
      <span />
    </div>
  );
}

function RootKeyRow({
  rootKey,
  selected,
  onEditKey,
}: {
  rootKey: RootKey;
  selected: boolean;
  onEditKey: (rootKey: RootKey) => void;
}) {
  const name = rootKey.name ?? "Unnamed Root Key";

  return (
    <ResourceListItem
      className={cn(
        "grid h-12 grid-cols-[minmax(0,1.1fr)_160px_minmax(0,1.3fr)_minmax(0,0.8fr)_minmax(0,0.8fr)_32px] items-center gap-4 px-4 text-xs text-gray-12 transition-colors hover:bg-grayA-2",
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
          <IconKey2Outline12 className="text-gray-12" />
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
      <span className="min-w-0 truncate font-mono text-xs text-gray-11">
        {rootKey.start}••••{rootKey.end}
      </span>
      <span className="min-w-0">
        <PermissionsCell permissions={rootKey.permissions} />
      </span>
      <span className="relative z-10 min-w-0">
        <LastUsedCell lastUsedAt={rootKey.lastUsedAt} isSelected={selected} />
      </span>
      <TimestampInfo
        value={rootKey.createdAt}
        displayType="relative"
        side="top"
        align="center"
        className="relative z-10 min-w-0 justify-self-start truncate text-sm leading-5 text-gray-9"
      />
      <span className="relative z-10 flex justify-end">
        <RootKeysTableActions rootKey={rootKey} onEditKey={onEditKey} />
      </span>
    </ResourceListItem>
  );
}

function RootKeysTableSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading Root Keys...</output>
      <div className="overflow-x-auto">
        <div className="min-w-[960px]">
          <RootKeysTableHeader />
          <ResourceListBody aria-hidden="true">
            {SKELETON_KEYS.map((key) => (
              <ResourceListItem
                key={key}
                className="grid h-12 grid-cols-[minmax(0,1.1fr)_160px_minmax(0,1.3fr)_minmax(0,0.8fr)_minmax(0,0.8fr)_32px] items-center gap-4 px-4"
              >
                <span className="flex items-center gap-2.5">
                  <Skeleton className="size-6 shrink-0 rounded-md" />
                  <Skeleton className="h-3 w-28" />
                </span>
                <Skeleton className="h-3 w-32" />
                <Skeleton className="h-3 w-36" />
                <Skeleton className="h-5 w-24 rounded-md" />
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
