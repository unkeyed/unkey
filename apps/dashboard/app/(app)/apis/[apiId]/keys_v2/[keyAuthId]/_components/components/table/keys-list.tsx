"use client";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import {
  BookBookmark,
  ChartActivity2,
  CircleHalfDottedClock,
  CircleLock,
  Key,
} from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { useKeysListQuery } from "./hooks/use-logs-query";
import { getRowClassName, STATUS_STYLES } from "./utils/get-row-class";
import { TimestampInfo } from "@/components/timestamp-info";
import { cn } from "@unkey/ui/src/lib/utils";

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
                {key.name && (
                  <span className="font-sans text-accent-9">{key.name}</span>
                )}
              </div>
              {!key.enabled && (
                <Badge className="px-1.5 rounded-md bg-warningA-3 text-warning-11 text-xs">
                  Disabled
                </Badge>
              )}
            </div>
          </div>
        ),
      },
      {
        key: "identitiy",
        header: "Identity",
        width: "10%",
        render: (key) => (
          <div>{key.identity?.external_id ?? key.owner_id ?? "—"}</div>
        ),
      },
      {
        key: "value",
        header: "Value",
        width: "15%",
        render: (key) => <HiddenValueCell value={key.start} title="Value" />,
      },
      {
        key: "usage",
        header: "Usage",
        width: "15%",
        render: () => <SimpleBarChart />,
      },
      {
        key: "last_used",
        header: "Last Used",
        width: "15%",
        render: (key) => {
          return (
            <Badge
              className={cn(
                "px-1.5 rounded-md bg-grayA-3 text-grayA-11 flex gap-2 items-center w-24",
                selectedKey?.id === key.id
                  ? STATUS_STYLES.badge.selected
                  : STATUS_STYLES.badge.default
              )}
            >
              <div>
                <ChartActivity2 size="sm-regular" />
              </div>
              <div className="truncate">5 days ago</div>
            </Badge>
          );
        },
      },
      {
        key: "expires",
        header: "Expires In",
        width: "15%",
        render: (key) => {
          // Function to check if timestamp is expired or close to expiry
          const getExpiryStatus = (expiryTimestamp: number | null) => {
            if (!expiryTimestamp) {
              return "valid"; // No expiration
            }

            const now = Date.now();

            // If already expired
            if (expiryTimestamp <= now) {
              return "expired";
            }

            const thirtyMinutesInMs = 30 * 60 * 1000;
            if (expiryTimestamp - now <= thirtyMinutesInMs) {
              return "expiring-soon";
            }

            return "valid";
          };

          const expiryStatus = getExpiryStatus(key.expires);

          return (
            <Badge
              className={cn(
                "px-1.5 rounded-md flex gap-2 items-center w-[135px] border",
                selectedKey?.id === key.id
                  ? STATUS_STYLES.badge.selected
                  : expiryStatus === "expired" ||
                    expiryStatus === "expiring-soon"
                  ? "bg-orange-3 text-orange-11 group-hover:bg-orange-4"
                  : STATUS_STYLES.badge.default
              )}
            >
              <div>
                <CircleHalfDottedClock size="sm-regular" />
              </div>
              {expiryStatus === "expired" ? (
                "Expired"
              ) : key.expires ? (
                <TimestampInfo value={key.expires} className="truncate" />
              ) : (
                <div>—</div>
              )}
            </Badge>
          );
        },
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
                No key verification data to show. Once requests are made with
                API keys, you'll see a summary of successful and failed
                verification attempts.
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

const HiddenValueCell = ({
  value,
  title = "Value",
}: {
  value: string;
  title: string;
}) => {
  const [isHovered, setIsHovered] = useState(false);
  // Show only first 4 characters, then dots
  const displayValue = isHovered
    ? value
    : `${value.substring(0, 2)}${
        value.length > 2 ? "•".repeat(Math.min(10, value.length - 2)) : ""
      }`;

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard
      .writeText(value)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    <>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
      <div
        className="rounded-lg border bg-grayA-1 border-accent-4 text-grayA-11 w-[150px] px-2 py-1 flex gap-2 items-center cursor-pointer"
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        onClick={(e) => handleClick(e)}
      >
        <div>
          <CircleLock size="sm-regular" className="text-gray-9" />
        </div>
        <div className="truncate">{displayValue}</div>
      </div>
    </>
  );
};

const SimpleBarChart = () => {
  // Generate random heights for bars (between 5 and 18px)
  const generateBars = () => {
    const bars = [];
    const numBars = 30; // Number of bars to display

    for (let i = 0; i < numBars; i++) {
      const totalHeight = Math.floor(Math.random() * 14) + 5; // 5-18px height
      const topHeight = Math.floor(Math.random() * 4) + 1; // 1-4px for orange top
      const bottomHeight = totalHeight - topHeight;

      bars.push({
        id: i,
        totalHeight,
        topHeight,
        bottomHeight,
      });
    }

    return bars;
  };

  const bars = generateBars();

  return (
    <div className="flex items-end h-[28px] bg-gray-1 w-fit border border-transparent hover:border-grayA-3  group-hover:border-grayA-3 rounded-[6px_6px_5px_5px] border-inside px-1 py-0">
      {bars.map((bar) => (
        <div
          key={bar.id}
          className="flex flex-col"
          style={{ marginRight: "2px" }}
        >
          <div
            className="w-[3px] bg-error-9"
            style={{ height: `${bar.topHeight}px` }}
          />
          <div
            className="w-[3px] bg-grayA-5"
            style={{ height: `${bar.bottomHeight}px` }}
          />
        </div>
      ))}
    </div>
  );
};
