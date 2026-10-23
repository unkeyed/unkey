"use client";
import {
  createApiKeyColumns,
  getRowClassName,
  renderApiKeySkeletonRow,
  useApiKeysListQuery,
} from "@/components/api-keys-table";
import { SelectionControls } from "@/components/api-keys-table/components/selection-controls";
import { LoadError } from "@/components/load-error";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { IconBookBookmarkOutline18, IconKeyOutline18 } from "@unkey/icons";
import {
  DataTable,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PaginationFooter,
  buttonVariants,
} from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";

const TABLE_CONFIG = {
  rowHeight: 52,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

export const KeysList = ({
  keyspaceId,
  apiId,
}: {
  keyspaceId: string;
  apiId: string;
}) => {
  const workspace = useWorkspaceNavigation();

  const {
    state,
    keys,
    isInitialLoading,
    isNavigating,
    onPageChange,
    page,
    hasNextPage,
    sorting,
    onSortingChange,
  } = useApiKeysListQuery({ keyAuthId: keyspaceId });

  const [selectedKey, setSelectedKey] = useState<KeyDetails | null>(null);
  const [navigatingKeyId, setNavigatingKeyId] = useState<string | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());

  const handlePageChange = useCallback(
    (newPage: number) => {
      setSelectedKeys(new Set());
      setNavigatingKeyId(null);
      onPageChange(newPage);
    },
    [onPageChange],
  );

  const handleSortingChange = useCallback(
    (updater: Parameters<typeof onSortingChange>[0]) => {
      setSelectedKeys(new Set());
      setNavigatingKeyId(null);
      onSortingChange(updater);
    },
    [onSortingChange],
  );

  const handleRowClick = useCallback((key: KeyDetails | null) => {
    setSelectedKey(key);
  }, []);

  const toggleSelection = useCallback((keyId: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(keyId)) {
        next.delete(keyId);
      } else {
        next.add(keyId);
      }
      return next;
    });
  }, []);

  const handleNavigateToKey = useCallback((keyId: string) => {
    setNavigatingKeyId(keyId);
    setSelectedKey(null);
  }, []);

  const getRowClassNameMemoized = useCallback(
    (key: KeyDetails) => getRowClassName(key, selectedKey),
    [selectedKey],
  );

  const getSelectedKeysState = useCallback(() => {
    if (selectedKeys.size === 0) {
      return null;
    }

    let allEnabled = true;
    let allDisabled = true;

    for (const keyId of selectedKeys) {
      const key = keys.find((k) => k.id === keyId);
      if (!key) {
        continue;
      }

      if (key.enabled) {
        allDisabled = false;
      } else {
        allEnabled = false;
      }

      if (!allEnabled && !allDisabled) {
        break;
      }
    }

    if (allEnabled) {
      return "all-enabled";
    }
    if (allDisabled) {
      return "all-disabled";
    }
    return "mixed";
  }, [selectedKeys, keys]);

  const columns = useMemo(
    () =>
      createApiKeyColumns({
        keyspaceId,
        apiId,
        workspaceSlug: workspace?.slug,
        selectedKeyId: selectedKey?.id ?? null,
        navigatingKeyId,
        selectedKeys,
        onToggleSelection: toggleSelection,
        onNavigateToKey: handleNavigateToKey,
      }),
    [
      keyspaceId,
      apiId,
      workspace?.slug,
      selectedKey?.id,
      navigatingKeyId,
      selectedKeys,
      toggleSelection,
      handleNavigateToKey,
    ],
  );

  if (state.status === "failed") {
    return <LoadError title="Could not load keys" error={state.error} onRetry={state.retry} />;
  }

  return (
    <>
      <DataTable
        data={keys}
        columns={columns}
        getRowId={(key) => key.id}
        isLoading={isInitialLoading}
        onRowClick={handleRowClick}
        selectedItem={selectedKey}
        rowClassName={getRowClassNameMemoized}
        renderSkeletonRow={renderApiKeySkeletonRow}
        sorting={sorting}
        onSortingChange={handleSortingChange}
        config={TABLE_CONFIG}
        emptyState={
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconKeyOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No API Keys Found</EmptyStateTitle>
              <EmptyStateDescription>
                There are no API keys associated with this service yet. Create your first API key to
                get started.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
                className={buttonVariants({ variant: "outline", size: "md" })}
              >
                <IconBookBookmarkOutline18 />
                Learn about Keys
              </a>
            </EmptyStateActions>
          </EmptyState>
        }
      />
      <PaginationFooter
        mode="cursor"
        page={page}
        rowCount={keys.length}
        hasNextPage={hasNextPage}
        onPageChange={handlePageChange}
        itemLabel={keys.length === 1 ? "key" : "keys"}
        loading={isInitialLoading}
        disabled={isNavigating}
        headerContent={
          <SelectionControls
            selectedKeys={selectedKeys}
            setSelectedKeys={setSelectedKeys}
            keys={keys}
            getSelectedKeysState={getSelectedKeysState}
          />
        }
      />
    </>
  );
};
