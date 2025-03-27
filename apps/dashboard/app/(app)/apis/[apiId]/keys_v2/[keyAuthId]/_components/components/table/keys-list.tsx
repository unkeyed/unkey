"use client";
import { Badge } from "@/components/ui/badge";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark, Key } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { useKeysListQuery } from "./hooks/use-logs-query";
import { getRowClassName } from "./utils/get-row-class";

type Props = {
  keyspaceId: string;
};

export const KeysList: React.FC<Props> = ({ keyspaceId }) => {
  const { keys, isLoading, isLoadingMore, loadMore } = useKeysListQuery({
    keyAuthId: keyspaceId,
  });
  const [selectedKey, setSelectedKey] = useState<KeyDetails | null>(null);

  const columns = (): Column<KeyDetails>[] => {
    return [
      {
        key: "key",
        header: "Key",
        width: "25%",
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

                <span className="font-sans text-accent-9">{key.name}</span>
              </div>
            </div>
          </div>
        ),
      },
      {
        key: "status",
        header: "Status",
        width: "10%",
        render: (key) => <div>{!key.enabled && <Badge variant="secondary">Disabled</Badge>}</div>,
      },
      {
        key: "actions",
        header: "",
        width: "5%",
        render: () => (
          <div className="flex items-center justify-end">
            <Button variant="ghost">
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        ),
      },
    ];
  };

  return (
    <div className="flex flex-col gap-8 mb-20">
      <VirtualTable
        data={keys}
        isLoading={isLoading}
        isFetchingNextPage={isLoadingMore}
        onLoadMore={loadMore}
        columns={columns()}
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
