"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { BookBookmark, Focus, Key } from "@unkey/icons";
import { Button, Empty, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import React from "react";
import { VerificationBarChart } from "./components/bar-chart";
import { HiddenValueCell } from "./components/hidden-value";
import { LastUsedCell } from "./components/last-used";
import {
  KeyColumnSkeleton,
  LastUsedColumnSkeleton,
  StatusColumnSkeleton,
  UsageColumnSkeleton,
  ValueColumnSkeleton,
} from "./components/skeletons";
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
        render: (key) => {
          const identity = key.identity?.external_id ?? key.owner_id;
          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center",
                identity ? "bg-successA-3" : "bg-grayA-3",
              )}
            >
              {identity ? (
                <Focus size="md-regular" className="text-successA-11" />
              ) : (
                <Key size="md-regular" />
              )}
            </div>
          );

          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-4 items-center">
                {identity ? (
                  <TooltipProvider delayDuration={100}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        {React.cloneElement(iconContainer, {
                          className: cn(iconContainer.props.className, "cursor-pointer"),
                        })}
                      </TooltipTrigger>
                      <TooltipContent
                        className="bg-gray-1 px-4 py-2 border border-gray-4 shadow-md font-medium text-xs text-accent-12"
                        side="right"
                      >
                        This key is associated with the identity:{" "}
                        <span className="font-mono bg-gray-3 px-1 rounded">{identity}</span>
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                ) : (
                  iconContainer
                )}

                <div className="flex flex-col gap-1 text-xs">
                  <div className="font-mono font-medium truncate text-brand-12">
                    {key.id.substring(0, 8)}...
                    {key.id.substring(key.id.length - 4)}
                  </div>
                  {key.name && <span className="font-sans text-accent-9">{key.name}</span>}
                </div>
              </div>
            </div>
          );
        },
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
        renderSkeletonRow={({ columns, rowHeight }) =>
          columns.map((column, idx) => (
            <td
              key={column.key}
              className={cn(
                "text-xs align-middle whitespace-nowrap pr-4",
                idx === 0 ? "pl-[18px]" : "",
                column.key === "key" ? "py-[6px]" : "py-1",
              )}
              style={{ height: `${rowHeight}px` }}
            >
              {column.key === "key" && <KeyColumnSkeleton />}
              {column.key === "value" && <ValueColumnSkeleton />}
              {column.key === "usage" && <UsageColumnSkeleton />}
              {column.key === "last_used" && <LastUsedColumnSkeleton />}
              {column.key === "status" && <StatusColumnSkeleton />}
              {!["key", "value", "usage", "last_used", "status"].includes(column.key) && (
                <div className="h-4 w-full bg-grayA-3 rounded animate-pulse" />
              )}
            </td>
          ))
        }
      />
    </div>
  );
};

// const OverrideIndicator = ({ log, style }: OverrideIndicatorProps) => (
//   <TooltipProvider>
//     <Tooltip>
//       <TooltipTrigger asChild>
//         <div className="group relative p-3 cursor-pointer -ml-[5px]">
//           <div className="absolute inset-0" />
//           <div
//             className={cn(
//               "size-[6px] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2",
//               calculateBlockedPercentage(log)
//                 ? "bg-orange-10 hover:bg-orange-11"
//                 : "bg-warning-10"
//             )}
//           />
//         </div>
//       </TooltipTrigger>
//       <TooltipContent
//         className="bg-gray-1 px-4 py-2 border border-gray-4 shadow-md font-medium text-xs text-accent-12 w-[265px]"
//         side="right"
//       >
//         <div className="flex gap-3 items-center">
//           <div
//             className={cn(
//               style.badge.default,
//               "rounded p-1",
//               "bg-accent-4 text-accent-11 group-hover:bg-accent-5"
//             )}
//           >
//             <ArrowDotAntiClockwise size="md-regular" />
//           </div>
//           <div className="flex flex-col gap-1">
//             <div className="text-sm flex gap-[10px] items-center">
//               <span className="font-medium text-sm text-accent-12">
//                 Custom override in effect
//               </span>
//               <div className="size-[6px] rounded-full bg-warning-10" />
//             </div>
//             {log.override && (
//               <div className="text-accent-9">
//                 Limit set to{" "}
//                 <span className="text-accent-12">
//                   {formatNumber(log.override.limit)}{" "}
//                 </span>
//                 requests per{" "}
//                 <span className="text-accent-12">
//                   {ms(log.override.duration)}
//                 </span>
//               </div>
//             )}
//           </div>
//         </div>
//       </TooltipContent>
//     </Tooltip>
//   </TooltipProvider>
// );
