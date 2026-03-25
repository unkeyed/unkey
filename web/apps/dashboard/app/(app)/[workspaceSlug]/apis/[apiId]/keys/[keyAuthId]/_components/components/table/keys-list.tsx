"use client";
import {
  createApiKeyColumns,
  getRowClassName,
  renderApiKeySkeletonRow,
  useApiKeysListQuery,
} from "@/components/api-keys-table";
import { SelectionControls } from "@/components/api-keys-table/components/selection-controls";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, PaginationFooter } from "@unkey/ui";
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
    keys,
    isInitialLoading,
    isFetching,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
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
        workspaceSlug: workspace?.slug ?? "",
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

  const isNavigating = isFetching && !isInitialLoading;

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
        config={TABLE_CONFIG}
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No API Keys Found</Empty.Title>
              <Empty.Description className="text-left">
                There are no API keys associated with this service yet. Create your first API key to
                get started.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Learn about Keys
                  </Button>
                </a>
              </Empty.Actions>
            </Empty>
          </div>
        }
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={handlePageChange}
        itemLabel="keys"
        loading={isInitialLoading}
        disabled={isNavigating}
        headerContent={
          selectedKeys.size > 0 ? (
            <SelectionControls
              selectedKeys={selectedKeys}
              setSelectedKeys={setSelectedKeys}
              keys={keys}
              getSelectedKeysState={getSelectedKeysState}
            />
          ) : undefined
        }
      />
    </>
  );
};
