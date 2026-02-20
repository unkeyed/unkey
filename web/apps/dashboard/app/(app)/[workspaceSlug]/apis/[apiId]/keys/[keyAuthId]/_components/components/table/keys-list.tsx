"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { shortenId } from "@/lib/shorten-id";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark, Dots, Focus, Key } from "@unkey/icons";
import { Button, Checkbox, Empty, InfoTooltip, Loading } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import Link from "next/link";
import React, { useCallback, useMemo, useState } from "react";
import { VerificationBarChart } from "./components/bar-chart";
import { HiddenValueCell } from "./components/hidden-value";
import { LastUsedCell } from "./components/last-used";
import { SelectionControls } from "./components/selection-controls";
import {
  ActionColumnSkeleton,
  KeyColumnSkeleton,
  LastUsedColumnSkeleton,
  StatusColumnSkeleton,
  UsageColumnSkeleton,
  ValueColumnSkeleton,
} from "./components/skeletons";
import { StatusDisplay } from "./components/status-cell";
import { useKeysListQuery } from "./hooks/use-keys-list-query";
import { getRowClassName } from "./utils/get-row-class";

const KeysTableActionPopover = dynamic(
  () =>
    import("./components/actions/keys-table-action.popover.constants").then(
      (mod) => mod.KeysTableActions,
    ),
  {
    ssr: false,
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

export const KeysList = ({
  keyspaceId,
  apiId,
}: {
  keyspaceId: string;
  apiId: string;
}) => {
  const workspace = useWorkspaceNavigation();

  const { keys, isLoading, isLoadingMore, loadMore, totalCount, hasMore } = useKeysListQuery({
    keyAuthId: keyspaceId,
  });
  const [selectedKey, setSelectedKey] = useState<KeyDetails | null>(null);
  const [navigatingKeyId, setNavigatingKeyId] = useState<string | null>(null);
  // Add state for selected keys
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  // Track which row is being hovered
  const [hoveredKeyId, setHoveredKeyId] = useState<string | null>(null);
  const handleLinkClick = useCallback((keyId: string) => {
    setNavigatingKeyId(keyId);
    setSelectedKey(null);
  }, []);

  const toggleSelection = useCallback((keyId: string) => {
    setSelectedKeys((prevSelected) => {
      const newSelected = new Set(prevSelected);
      if (newSelected.has(keyId)) {
        newSelected.delete(keyId);
      } else {
        newSelected.add(keyId);
      }
      return newSelected;
    });
  }, []);

  const columns: Column<KeyDetails>[] = useMemo(
    () => [
      {
        key: "key",
        header: "Key",
        width: "20%", // Increased from 10% to accommodate key info
        headerClassName: "pl-[18px]",
        render: (key) => {
          const identity = key.identity?.external_id ?? key.owner_id;
          const isNavigating = key.id === navigatingKeyId;
          const isSelected = selectedKeys.has(key.id);
          const isHovered = hoveredKeyId === key.id;

          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded-sm flex items-center justify-center cursor-pointer",
                identity ? "bg-successA-3" : "bg-grayA-3",
                isSelected && "bg-brand-5",
              )}
              onMouseEnter={() => setHoveredKeyId(key.id)}
              onMouseLeave={() => setHoveredKeyId(null)}
            >
              {isNavigating ? (
                <div className={cn(identity ? "text-successA-11" : "text-grayA-11")}>
                  <Loading size={18} />
                </div>
              ) : (
                <>
                  {/* Show icon when not selected and not hovered */}
                  {!isSelected && !isHovered && (
                    // biome-ignore lint/complexity/noUselessFragments: <explanation>
                    <>
                      {identity ? (
                        <Focus iconSize="md-medium" className="text-successA-11" />
                      ) : (
                        <Key iconSize="md-medium" />
                      )}
                    </>
                  )}

                  {/* Show checkbox when selected or hovered */}
                  {(isSelected || isHovered) && (
                    <Checkbox
                      checked={isSelected}
                      className="size-4 [&_svg]:size-3"
                      onCheckedChange={() => toggleSelection(key.id)}
                    />
                  )}
                </>
              )}
            </div>
          );

          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-4 items-center">
                {identity ? (
                  <InfoTooltip
                    delayDuration={100}
                    variant="muted"
                    position={{ side: "right" }}
                    className="bg-gray-1 px-4 py-2 border border-gray-4 shadow-md font-medium text-xs text-accent-12"
                    content={
                      <>
                        This key is associated with the identity:{" "}
                        {key.identity_id && workspace ? (
                          <Link
                            title={"View details for identity"}
                            className="font-mono group-hover:underline decoration-dotted"
                            href={`/${workspace.slug}/identities/${key.identity_id}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            aria-disabled={isNavigating}
                          >
                            <span className="font-mono bg-gray-4 p-1 rounded-sm">{identity}</span>
                          </Link>
                        ) : (
                          <span className="font-mono bg-gray-4 p-1 rounded-sm">{identity}</span>
                        )}
                      </>
                    }
                    asChild
                  >
                    {React.cloneElement(iconContainer, {
                      className: cn(iconContainer.props.className, "cursor-pointer"),
                    })}
                  </InfoTooltip>
                ) : (
                  iconContainer
                )}

                <div className="flex flex-col gap-1 text-xs">
                  <Link
                    title={`View details for ${key.id}`}
                    className="font-mono group-hover:underline decoration-dotted"
                    href={`/${workspace.slug}/apis/${apiId}/keys/${keyspaceId}/${key.id}`}
                    aria-disabled={isNavigating}
                    onClick={() => {
                      handleLinkClick(key.id);
                    }}
                  >
                    <div className="font-mono font-medium truncate text-brand-12">
                      {shortenId(key.id)}
                    </div>
                  </Link>
                  {key.name && (
                    <span
                      className="font-sans text-accent-9 truncate max-w-[120px]"
                      title={key.name}
                    >
                      {key.name}
                    </span>
                  )}
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "value",
        header: "Value",
        width: "15%", // Keep at 15%
        render: (key) => (
          <HiddenValueCell value={key.start} title="Value" selected={selectedKey?.id === key.id} />
        ),
      },
      {
        key: "usage",
        header: "Usage in last 36h",
        width: "20%", // Increased from 15% for better chart display
        render: (key) => (
          <VerificationBarChart
            keyAuthId={keyspaceId}
            keyId={key.id}
            selected={key.id === selectedKey?.id}
          />
        ),
      },
      {
        key: "last_used",
        header: "Last Used",
        width: "15%", // Keep at 15%
        render: (key) => {
          return (
            <LastUsedCell
              keyId={key.id}
              keyAuthId={keyspaceId}
              isSelected={selectedKey?.id === key.id}
            />
          );
        },
      },
      {
        key: "status",
        header: "Status",
        width: "15%", // Keep at 15%
        render: (key) => {
          return (
            <StatusDisplay
              keyData={key}
              keyAuthId={keyspaceId}
              isSelected={selectedKey?.id === key.id}
            />
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "15%", // Keep at 15%
        render: (key) => {
          return <KeysTableActionPopover keyData={key} />;
        },
      },
    ],
    [
      keyspaceId,
      selectedKey?.id,
      apiId,
      navigatingKeyId,
      handleLinkClick,
      selectedKeys,
      toggleSelection,
      hoveredKeyId,
      workspace,
    ],
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

      // Early exit if we already found a mix
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

  return (
    <>
      <VirtualTable
        data={keys}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns}
        onRowClick={setSelectedKey}
        selectedItem={selectedKey}
        keyExtractor={(log) => log.id}
        rowClassName={(log) => getRowClassName(log, selectedKey as KeyDetails)}
        loadMoreFooterProps={{
          hide: isLoading,
          buttonText: "Load more keys",
          hasMore,
          headerContent: (
            <SelectionControls
              selectedKeys={selectedKeys}
              setSelectedKeys={setSelectedKeys}
              keys={keys}
              getSelectedKeysState={getSelectedKeysState}
            />
          ),
          countInfoText: (
            <div className="flex gap-2">
              <span>Showing</span>{" "}
              <span className="text-accent-12">{new Intl.NumberFormat().format(keys.length)}</span>
              <span>of</span>
              {new Intl.NumberFormat().format(totalCount)}
              <span>keys</span>
            </div>
          ),
        }}
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
                column.key === "key" ? "py-[6px]" : "py-1",
                column.cellClassName,
              )}
              style={{ height: `${rowHeight}px` }}
            >
              {column.key === "key" && <KeyColumnSkeleton />}
              {column.key === "value" && <ValueColumnSkeleton />}
              {column.key === "usage" && <UsageColumnSkeleton />}
              {column.key === "last_used" && <LastUsedColumnSkeleton />}
              {column.key === "status" && <StatusColumnSkeleton />}
              {column.key === "action" && <ActionColumnSkeleton />}
              {!["select", "key", "value", "usage", "last_used", "status", "action"].includes(
                column.key,
              ) && <div className="h-4 w-full bg-grayA-3 rounded-sm animate-pulse" />}
            </td>
          ))
        }
      />
    </>
  );
};
