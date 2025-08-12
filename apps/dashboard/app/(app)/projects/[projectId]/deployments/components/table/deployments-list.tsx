"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import {
  ArrowDotAntiClockwise,
  Ban,
  BookBookmark,
  CircleCheck,
  CircleDotted,
  CircleHalfDottedClock,
  CircleWarning,
  ClockRotateClockwise,
  Cloud,
  CloudUp,
  HalfDottedCirclePlay,
  Nut,
} from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  KeyColumnSkeleton,
  LastUpdatedColumnSkeleton,
  PermissionsColumnSkeleton,
  RootKeyColumnSkeleton,
} from "./components/skeletons";
import { getRowClassName } from "./utils/get-row-class";
import { useDeploymentsListQuery } from "./hooks/use-deployments-list-query";
import type { Deployment } from "@/lib/trpc/routers/deploy/project/deployment/list";
import { shortenId } from "@/lib/shorten-id";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";

// const RootKeysTableActions = dynamic(
//   () =>
//     import(
//       "./components/actions/root-keys-table-action.popover.constants"
//     ).then((mod) => mod.RootKeysTableActions),
//   {
//     loading: () => (
//       <button
//         type="button"
//         className={cn(
//           "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
//           "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7"
//         )}
//       >
//         <Dots
//           className="group-hover:text-gray-12 text-gray-11"
//           size="sm-regular"
//         />
//       </button>
//     ),
//   }
// );

export const DeploymentsList = () => {
  const {
    deployments,
    isLoading,
    isLoadingMore,
    loadMore,
    totalCount,
    hasMore,
  } = useDeploymentsListQuery();
  const [selectedDeployment, setSelectedDeployment] =
    useState<Deployment | null>(null);

  const columns: Column<Deployment>[] = useMemo(
    () => [
      {
        key: "deployment_id",
        header: "Deployment ID",
        width: "15%",
        headerClassName: "pl-[18px]",
        render: (deployment) => {
          const isSelected = deployment.id === selectedDeployment?.id;
          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5"
              )}
            >
              <Cloud size="sm-regular" className="text-gray-12" />
            </div>
          );
          return (
            <div className="flex flex-col items-start px-[18px] py-[12px]">
              <div className="flex gap-5 items-center w-full">
                {iconContainer}
                <div className="w-[150px]">
                  <div
                    className={cn(
                      "font-normal font-mono truncate leading-5 text-[13px]",
                      "text-accent-12"
                    )}
                  >
                    {shortenId(deployment.id)}
                  </div>
                  <div
                    className={cn(
                      "font-normal font-mono truncate text-xs mt-1",
                      "text-gray-9"
                    )}
                  >
                    {deployment.pullRequest.title}
                  </div>
                </div>
              </div>
            </div>
          );
        },
      },

      {
        key: "status",
        header: "Status",
        width: "15%",
        headerClassName: "pl-[18px]",
        render: (deployment) => {
          return (
            <div className="px-[18px] py-[12px]">
              <DeploymentStatusBadge status={deployment.status} />
            </div>
          );
        },
      },
    ],
    [selectedDeployment?.id]
  );

  return (
    <VirtualTable
      data={deployments}
      isLoading={isLoading}
      isFetchingNextPage={isLoadingMore}
      onLoadMore={loadMore}
      columns={columns}
      onRowClick={setSelectedDeployment}
      selectedItem={selectedDeployment}
      keyExtractor={(deployment) => deployment.id}
      rowClassName={(deployment) =>
        getRowClassName(deployment, selectedDeployment)
      }
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more deployments",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>{" "}
            <span className="text-accent-12">{deployments.length}</span>
            <span>of</span>
            {totalCount}
            <span>deployments</span>
          </div>
        ),
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Deployments Found</Empty.Title>
            <Empty.Description className="text-left">
              There are no deployments yet. Push to your connected repository or
              trigger a manual deployment to get started.
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/deployments"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Learn about Deployments
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
        columns.map((column) => (
          <td
            key={column.key}
            className={cn(
              "text-xs align-middle whitespace-nowrap",
              column.key === "root_key" ? "py-[6px]" : "py-1"
            )}
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "root_key" && <RootKeyColumnSkeleton />}
            {column.key === "key" && <KeyColumnSkeleton />}
            {column.key === "created_at" && <CreatedAtColumnSkeleton />}
            {column.key === "permissions" && <PermissionsColumnSkeleton />}
            {column.key === "last_updated" && <LastUpdatedColumnSkeleton />}
            {column.key === "action" && <ActionColumnSkeleton />}
          </td>
        ))
      }
    />
  );
};
