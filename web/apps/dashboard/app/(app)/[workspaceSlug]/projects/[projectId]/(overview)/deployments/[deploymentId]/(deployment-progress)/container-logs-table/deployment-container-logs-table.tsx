"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import { cn } from "@/lib/utils";
import { type ContainerLogRow, containerLogColumns } from "./columns";
import { getContainerLogRowClass } from "./get-row-class";
import { MessageColumnSkeleton, SeverityColumnSkeleton, TimeColumnSkeleton } from "./skeletons";

type Props = {
  logs: ContainerLogRow[];
};

export const DeploymentContainerLogsTable: React.FC<Props> = ({ logs }) => {
  return (
    <VirtualTable
      data={logs}
      isLoading={logs.length === 0}
      columns={containerLogColumns}
      renderSkeletonRow={({ columns, rowHeight }) =>
        columns.map((column, idx) => (
          <td
            key={column.key}
            className={cn(
              "text-xs align-middle whitespace-nowrap",
              idx === 0 ? "pl-[25px]" : "",
              column.cellClassName,
            )}
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "time" && <TimeColumnSkeleton />}
            {column.key === "severity" && <SeverityColumnSkeleton />}
            {column.key === "message" && <MessageColumnSkeleton />}
          </td>
        ))
      }
      keyExtractor={(log) => log.time}
      rowClassName={(log) => getContainerLogRowClass(log)}
      fixedHeight={256}
      autoScrollToBottom
      config={{
        containerPadding: "px-0 py-0",
        className: "bg-transparent",
      }}
      emptyState={<div className="px-8 py-4 text-sm text-gray-9">No runtime logs yet</div>}
    />
  );
};
