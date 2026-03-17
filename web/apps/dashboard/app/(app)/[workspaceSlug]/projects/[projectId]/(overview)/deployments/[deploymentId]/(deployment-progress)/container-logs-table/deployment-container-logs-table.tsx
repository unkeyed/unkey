"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import { type ContainerLogRow, containerLogColumns } from "./columns";
import { getContainerLogRowClass } from "./get-row-class";
import { ContainerLogRowSkeleton } from "./skeletons";

type Props = {
  logs: ContainerLogRow[];
  isLoading: boolean;
};

export const DeploymentContainerLogsTable: React.FC<Props> = ({ logs, isLoading }) => {
  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      columns={containerLogColumns}
      renderSkeletonRow={({ columns, rowHeight }) =>
        columns.map((column) => (
          <td
            key={column.key}
            className="text-xs align-middle whitespace-nowrap"
            style={{ height: `${rowHeight}px` }}
          >
            <ContainerLogRowSkeleton />
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
      emptyState={<div className="px-8 py-4 text-sm text-gray-11">No runtime logs yet</div>}
    />
  );
};
