"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/components/ui/toaster";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark, ChartActivity2, CircleHalfDottedClock, CircleLock, Key } from "@unkey/icons";
import { Button, Empty, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import {
  type ProcessedTimeseriesDataPoint,
  useFetchVerificationTimeseries,
} from "./components/bar-chart/hooks/use-fetch-timeseries";
import { useKeysListQuery } from "./hooks/use-logs-query";
import { STATUS_STYLES, getRowClassName } from "./utils/get-row-class";

type Props = {
  keyspaceId: string;
};

export const KeysList = ({ keyspaceId }: Props) => {
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
                  : STATUS_STYLES.badge.default,
                (expiryStatus === "expired" || expiryStatus === "expiring-soon") &&
                  "bg-orange-4 text-orange-11 group-hover:bg-orange-4",
              )}
            >
              <div>
                <CircleHalfDottedClock size="sm-regular" />
              </div>
              {expiryStatus === "expired" ? (
                "Expired"
              ) : key.expires ? (
                <TimestampInfo
                  value={key.expires}
                  className="truncate capitalize"
                  displayType="relative"
                />
              ) : (
                <div>—</div>
              )}
            </Badge>
          );
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

const HiddenValueCell = ({
  value,
  title = "Value",
  selected,
}: {
  value: string;
  title: string;
  selected: boolean;
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
        className={cn(
          "rounded-lg border bg-grayA-1 border-accent-4 text-grayA-11 w-[150px] px-2 py-1 flex gap-2 items-center cursor-pointer h-[28px] group-hover:border-grayA-3",
          selected && "border-grayA-3",
        )}
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

type BarData = {
  id: string | number;
  topHeight: number;
  bottomHeight: number;
  totalHeight: number;
};

type VerificationBarChartProps = {
  keyAuthId: string;
  keyId: string;
  maxBars?: number;
  selected: boolean;
};

const MAX_HEIGHT_BUFFER_FACTOR = 1.3;
const MAX_BAR_HEIGHT = 28;
const VerificationBarChart = ({
  keyAuthId,
  keyId,
  selected,
  maxBars = 30,
}: VerificationBarChartProps) => {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(keyAuthId, keyId);

  const isEmpty = useMemo(
    () => timeseries.reduce((acc, crr) => acc + crr.total, 0) === 0,
    [timeseries],
  );

  const bars = useMemo((): BarData[] => {
    if (isLoading || isError || timeseries.length === 0) {
      // Return empty data if loading, error, or no data
      return Array(maxBars).fill({
        id: 0,
        topHeight: 0,
        bottomHeight: 0,
        totalHeight: 0,
      });
    }
    // Get the most recent data points (or all if less than maxBars)
    const recentData = timeseries.slice(-maxBars);
    // Calculate the maximum total value to normalize heights
    const maxTotal =
      Math.max(...recentData.map((item) => item.total), 1) * MAX_HEIGHT_BUFFER_FACTOR;
    // Generate bars from the data
    return recentData.map((item, index): BarData => {
      // Scale to fit within max height of 28px
      const totalHeight = Math.min(
        Math.round((item.total / maxTotal) * MAX_BAR_HEIGHT),
        MAX_BAR_HEIGHT,
      );
      // Calculate heights proportionally
      const topHeight = item.error
        ? Math.max(Math.round((item.error / item.total) * totalHeight), 1)
        : 0;
      const bottomHeight = Math.max(totalHeight - topHeight, 0);
      return {
        id: index,
        totalHeight,
        topHeight,
        bottomHeight,
      };
    });
  }, [timeseries, isLoading, isError, maxBars]);

  // Pad with empty bars if we have fewer than maxBars data points
  const displayBars = useMemo((): BarData[] => {
    const result = [...bars];
    while (result.length < maxBars) {
      result.unshift({
        id: `empty-${result.length}`,
        topHeight: 0,
        bottomHeight: 0,
        totalHeight: 0,
      });
    }
    return result;
  }, [bars, maxBars]);

  // Loading state - animated pulse effect for bars with grid layout
  if (isLoading) {
    return (
      <div
        // We need 156px when you calculate the gaps and height but we need some breathing space so its "158px" now
        className={cn(
          "grid items-end h-[28px] bg-gray-1 w-[158px] border border-transparent hover:border-grayA-3 group-hover:border-grayA-3 rounded-lg border-inside px-1 py-0 overflow-hidden",
          selected ? "border-grayA-3" : "border-transparent",
        )}
        style={{
          gridTemplateColumns: `repeat(${maxBars}, 3px)`,
          gap: "2px",
        }}
      >
        {Array(maxBars)
          .fill(0)
          .map((_, index) => (
            <div
              key={`loading-${
                // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                index
              }`}
              className="flex flex-col"
            >
              <div
                className="w-[3px] bg-error-9 animate-pulse"
                style={{ height: `${2 + Math.floor(Math.random() * 10)}px` }}
              />
              <div
                className="w-[3px] bg-grayA-5 animate-pulse"
                style={{ height: `${3 + Math.floor(Math.random() * 12)}px` }}
              />
            </div>
          ))}
      </div>
    );
  }

  // Error state with grid layout
  if (isError) {
    return (
      <div
        className={cn(
          "grid items-end h-[28px] bg-gray-1 w-[158px] border border-transparent hover:border-grayA-3 group-hover:border-grayA-3 rounded-lg border-inside px-1 py-0 overflow-hidden",
          selected ? "border-grayA-3" : "border-transparent",
        )}
        style={{
          gridTemplateColumns: "1fr",
        }}
      >
        <div className="flex items-center justify-center w-full h-full text-xs text-error-9 px-2">
          Error loading data
        </div>
      </div>
    );
  }

  // Empty state with grid layout
  if (isEmpty) {
    return (
      <div
        className={cn(
          "grid items-end h-[28px] bg-gray-1 w-[158px] border border-transparent hover:border-grayA-3 group-hover:border-grayA-3 rounded-lg border-inside px-1 py-0 overflow-hidden",
          selected ? "border-grayA-3" : "border-transparent",
        )}
        style={{
          gridTemplateColumns: "1fr",
        }}
      >
        <div className="flex items-center justify-center w-full h-full text-xs text-grayA-9 px-2">
          No data available
        </div>
      </div>
    );
  }

  // Data display with grid layout
  return (
    <PopoverDemo timeseries={timeseries}>
      <div
        className={cn(
          "grid items-end h-[28px] bg-gray-1 w-[158px] border border-transparent hover:border-grayA-3 group-hover:border-grayA-3 rounded-lg border-inside px-1 py-0 overflow-hidden",
          selected ? "border-grayA-3" : "border-transparent",
        )}
        style={{
          gridTemplateColumns: `repeat(${maxBars}, 3px)`,
          gap: "2px",
        }}
      >
        {displayBars.map((bar) => (
          <div key={bar.id} className="flex flex-col">
            <div className="w-[3px] bg-error-9" style={{ height: `${bar.topHeight}px` }} />
            <div className="w-[3px] bg-grayA-5" style={{ height: `${bar.bottomHeight}px` }} />
          </div>
        ))}
      </div>
    </PopoverDemo>
  );
};

export const LastUsedCell = ({
  keyAuthId,
  keyId,
  isSelected,
}: {
  keyAuthId: string;
  keyId: string;
  isSelected: boolean;
}) => {
  const { data, isLoading, isError } = trpc.api.keys.latestVerification.useQuery({
    keyAuthId,
    keyId,
  });

  return (
    <Badge
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center w-[140px]",
        isError
          ? "bg-error-3 text-error-11 border border-error-5"
          : isSelected
            ? STATUS_STYLES.badge.selected
            : STATUS_STYLES.badge.default,
      )}
    >
      <div>
        <ChartActivity2 size="sm-regular" />
      </div>
      <div className="truncate">
        {isLoading ? (
          <div className="flex items-center w-full space-x-1">
            <div className="h-2 w-2 bg-grayA-5 rounded-full animate-pulse" />
            <div className="h-2 w-12 bg-grayA-5 rounded animate-pulse" />
            <div className="h-2 w-12 bg-grayA-5 rounded animate-pulse" />
          </div>
        ) : isError ? (
          "Failed to load"
        ) : data?.lastVerificationTime ? (
          <TimestampInfo value={data.lastVerificationTime} className="truncate" />
        ) : (
          "Never used"
        )}
      </div>
    </Badge>
  );
};

interface PopoverDemoProps {
  children: React.ReactNode;
  timeseries: ProcessedTimeseriesDataPoint[];
}

type ErrorType = {
  type: string;
  value: number;
  color: string;
};

export function PopoverDemo({ children, timeseries }: PopoverDemoProps): JSX.Element {
  // Aggregate all timeseries data for the tooltip
  const aggregatedData = useMemo(() => {
    if (!timeseries || timeseries.length === 0) {
      return {
        valid: 0,
        rate_limited: 0,
        insufficient_permissions: 0,
        forbidden: 0,
        disabled: 0,
        expired: 0,
        usage_exceeded: 0,
        total: 0,
      };
    }

    return timeseries.reduce(
      (acc, dataPoint) => {
        acc.valid += dataPoint.valid || 0;
        acc.rate_limited += dataPoint.rate_limited || 0;
        acc.insufficient_permissions += dataPoint.insufficient_permissions || 0;
        acc.forbidden += dataPoint.forbidden || 0;
        acc.disabled += dataPoint.disabled || 0;
        acc.expired += dataPoint.expired || 0;
        acc.usage_exceeded += dataPoint.usage_exceeded || 0;
        acc.total += dataPoint.total || 0;
        return acc;
      },
      {
        valid: 0,
        rate_limited: 0,
        insufficient_permissions: 0,
        forbidden: 0,
        disabled: 0,
        expired: 0,
        usage_exceeded: 0,
        total: 0,
      },
    );
  }, [timeseries]);

  const errorTypes = useMemo(() => {
    return [
      {
        type: "Insufficient Permissions",
        value: formatNumber(aggregatedData.insufficient_permissions) || 0,
        color: "bg-error-9",
      },
      {
        type: "Rate Limited",
        value: formatNumber(aggregatedData.rate_limited) || 0,
        color: "bg-error-9",
      },
      {
        type: "Forbidden",
        value: formatNumber(aggregatedData.forbidden) || 0,
        color: "bg-error-9",
      },
      {
        type: "Disabled",
        value: formatNumber(aggregatedData.disabled) || 0,
        color: "bg-error-9",
      },
      {
        type: "Expired",
        value: formatNumber(aggregatedData.expired) || 0,
        color: "bg-error-9",
      },
      {
        type: "Usage Exceeded",
        value: formatNumber(aggregatedData.usage_exceeded) || 0,
        color: "bg-error-9",
      },
    ].filter((error) => Number(error.value) > 0) as ErrorType[];
  }, [aggregatedData]);

  return (
    <TooltipProvider>
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <div>{children}</div>
        </TooltipTrigger>
        <TooltipContent
          className="min-w-64 bg-gray-1 dark:bg-black shadow-2xl p-0 border border-grayA-2 rounded-lg overflow-hidden flex justify-start px-4 pt-2 pb-1 flex-col gap-1"
          side="bottom"
        >
          <div className="text-gray-12 font-medium text-[13px] pr-2">API Key Activity</div>
          <div className="text-xs text-grayA-9 pr-2 font-normal">Last 36 hours</div>

          {/* Valid count */}
          <div className="flex justify-between w-full items-center mt-3">
            <div className="flex gap-3 items-center">
              <div className="bg-gray-7 h-6 w-0.5 rounded-t rounded-b" />
              <div className="text-gray-12 font-medium text-[13px]">Valid</div>
            </div>
            <div className="text-gray-9 font-medium text-[13px]">
              {formatNumber(aggregatedData.valid)}
            </div>
          </div>

          <div className="mt-1" />

          {/* Error types */}
          <div className="flex flex-col">
            {errorTypes.map((error, index) => (
              <div
                // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                key={index}
                className="flex justify-between w-full items-center"
              >
                <div className="flex gap-3 items-center">
                  <div className={`${error.color} h-6 w-0.5 rounded-t rounded-b`} />
                  <div className="text-gray-12 font-medium text-[13px]">{error.type}</div>
                </div>
                <div className="text-gray-9 font-medium text-[13px]">{error.value}</div>
              </div>
            ))}

            {errorTypes.length === 0 && aggregatedData.valid === 0 && (
              <div className="text-gray-9 text-[13px] py-1">No verification activity</div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
