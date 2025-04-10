"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark, Key } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useMemo, useState } from "react";
import { VerificationBarChart } from "./components/bar-chart";
import { HiddenValueCell } from "./components/hidden-value";
import { LastUsedCell } from "./components/last-used";
import { StatusDisplay } from "./components/status-cell";
import { useKeysListQuery } from "./hooks/use-logs-query";
import { getRowClassName } from "./utils/get-row-class";

export const KeysList = ({ keyspaceId }: { keyspaceId: string }) => {
  const { keys, isLoading, isLoadingMore, loadMore } = useKeysListQuery({
    keyAuthId: keyspaceId,
  });
  const [selectedKey, setSelectedKey] = useState<KeyDetails | null>(null);

  const columns: Column<KeyDetails>[] = useMemo(
    () => [
      {
        key: "key",
        header: "Key",
        width: "10%",
        headerClassName: "pl-[18px]",
        render: (key) => (
          <div className="flex flex-col items-start px-[18px] py-[6px]">
            <div className="flex gap-4 items-center">
              <div className="bg-grayA-3 size-5 rounded flex items-center justify-center">
                <Key size="sm-regular" />
              </div>
              <div className="flex flex-col gap-1 text-xs">
                <div className="font-mono font-medium truncate text-brand-12">
                  {key.id.substring(0, 8)}...
                  {key.id.substring(key.id.length - 4)}
                </div>
                {key.name && <span className="font-sans text-accent-9">{key.name}</span>}
              </div>
            </div>
          </div>
        ),
      },
      {
        key: "value",
        header: "Value",
        width: "15%",
        render: (key) => (
          <HiddenValueCell value={key.start} title="Value" selected={selectedKey?.id === key.id} />
        ),
      },
      {
        key: "usage",
        header: "Usage",
        width: "15%",
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
        width: "15%",
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
        width: "auto",
        render: (key) => {
          return <StatusDisplay keyData={key} keyAuthId={keyspaceId} />;
        },
      },
    ],
    [keyspaceId, selectedKey?.id],
  );

  return (
    <div className="flex flex-col gap-8 mb-20">
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
        emptyState={
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>Key Verification Logs</Empty.Title>
              <Empty.Description className="text-left">
                No key verification data to show. Once requests are made with API keys, you'll see a
                summary of successful and failed verification attempts.
              </Empty.Description>
              <Empty.Actions className="mt-4 justify-start">
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Documentation
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
      />
    </div>
  );
};
