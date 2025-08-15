"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { shortenId } from "@/lib/shorten-id";
import type { Deployment } from "@/lib/trpc/routers/deploy/project/deployment/list";
import { BookBookmark, Cloud, CodeBranch, CodeCommit, Cube, Dots } from "@unkey/icons";
import { Button, Empty, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import { useMemo, useState } from "react";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { EnvStatusBadge } from "./components/env-status-badge";
import {
  ActionColumnSkeleton,
  AuthorColumnSkeleton,
  CreatedAtColumnSkeleton,
  DeploymentIdColumnSkeleton,
  EnvColumnSkeleton,
  InstancesColumnSkeleton,
  SourceColumnSkeleton,
  StatusColumnSkeleton,
} from "./components/skeletons";
import { useDeploymentsListQuery } from "./hooks/use-deployments-list-query";
import { getRowClassName } from "./utils/get-row-class";

const DeploymentListTableActions = dynamic(
  () =>
    import("./components/actions/deployment-list-table-action.popover.constants").then(
      (mod) => mod.DeploymentListTableActions,
    ),
  {
    loading: () => (
      <button
        type="button"
        className={cn(
          "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
          "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
        )}
      >
        <Dots className="group-hover:text-gray-12 text-gray-11" size="sm-regular" />
      </button>
    ),
  },
);

export const DeploymentsList = () => {
  const { deployments, isLoading, isLoadingMore, loadMore, totalCount, hasMore } =
    useDeploymentsListQuery();
  const [selectedDeployment, setSelectedDeployment] = useState<Deployment | null>(null);

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
                isSelected && "bg-grayA-5",
              )}
            >
              <Cloud size="sm-regular" className="text-gray-12" />
            </div>
          );
          return (
            <div className="flex flex-col items-start px-[18px] py-1.5">
              <div className="flex gap-5 items-center w-full">
                {iconContainer}
                <div className="w-[200px]">
                  <div className="flex items-center gap-2">
                    <div
                      className={cn(
                        "font-normal font-mono truncate leading-5 text-[13px]",
                        "text-accent-12",
                      )}
                    >
                      {shortenId(deployment.id)}
                    </div>
                    {deployment.environment === "production" && deployment.active && (
                      <EnvStatusBadge variant="current" text="Current" />
                    )}
                  </div>
                  <div className={cn("font-normal font-mono truncate text-xs mt-1", "text-gray-9")}>
                    {deployment.pullRequest.title}
                  </div>
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "env",
        header: "Environment",
        width: "10%",
        render: (deployment) => {
          return (
            <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit capitalize">
              {deployment.environment}
            </div>
          );
        },
      },
      {
        key: "status",
        header: "Status",
        width: "10%",
        render: (deployment) => {
          return <DeploymentStatusBadge status={deployment.status} />;
        },
      },
      {
        key: "instances",
        header: "Instances",
        width: "10%",
        render: (deployment) => {
          return (
            <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
              <Cube className="text-gray-12" size="sm-regular" />
              <div>
                <span className="font-semibold text-grayA-12 tabular-nums">
                  {deployment.instances}
                </span>
                <span>VM</span>
              </div>
            </div>
          );
        },
      },
      {
        key: "runtime",
        header: "Runtime",
        width: "10%",
        render: (deployment) => {
          return deployment.runtime ? (
            <div>
              <span className="font-semibold text-grayA-12 tabular-nums">{deployment.runtime}</span>
              <span>s</span>
            </div>
          ) : (
            "-"
          );
        },
      },
      {
        key: "size",
        header: "Size",
        width: "10%",
        render: (deployment) => {
          return (
            <div>
              <span className="font-semibold text-grayA-12 tabular-nums">{deployment.size}</span>
              <span>mb</span>
            </div>
          );
        },
      },
      {
        key: "source",
        header: "Source",
        width: "10%",
        render: (deployment) => {
          return (
            <div className="flex items-center gap-1">
              <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit">
                <CodeBranch className="text-gray-12" size="sm-regular" />
                <span className="tabular-nums">{deployment.source.branch}</span>
              </div>
              <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit shrink-0">
                <CodeCommit className="text-gray-12 rotate-90 shrink-0" size="md-bold" />
                <span className="tabular-nums">{deployment.source.gitSha}</span>
              </div>
            </div>
          );
        },
      },
      {
        key: "created_at",
        header: "Created at",
        width: "10%",
        render: (deployment) => {
          return (
            <TimestampInfo
              value={deployment.createdAt}
              className="font-mono group-hover:underline decoration-dotted"
            />
          );
        },
      },
      {
        key: "author",
        header: "Author",
        width: "10%",
        render: (deployment) => {
          return (
            <div className="flex items-center gap-2">
              <img src={deployment.author.image} alt="Author" className="rounded-full size-5" />
              <span className="font-medium text-grayA-12 text-xs">{deployment.author.name}</span>
            </div>
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "auto",
        render: (deployment) => {
          return <DeploymentListTableActions deployment={deployment} />;
        },
      },
    ],
    [selectedDeployment?.id],
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
      rowClassName={(deployment) => getRowClassName(deployment, selectedDeployment)}
      loadMoreFooterProps={{
        hide: isLoading,
        buttonText: "Load more deployments",
        hasMore,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span> <span className="text-accent-12">{deployments.length}</span>
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
              There are no deployments yet. Push to your connected repository or trigger a manual
              deployment to get started.
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
            className="text-xs align-middle whitespace-nowrap"
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "deployment_id" && <DeploymentIdColumnSkeleton />}
            {column.key === "env" && <EnvColumnSkeleton />}
            {column.key === "status" && <StatusColumnSkeleton />}
            {column.key === "instances" && <InstancesColumnSkeleton />}
            {column.key === "runtime" && <EnvColumnSkeleton />}
            {column.key === "size" && <EnvColumnSkeleton />}
            {column.key === "source" && <SourceColumnSkeleton />}
            {column.key === "created_at" && <CreatedAtColumnSkeleton />}
            {column.key === "author" && <AuthorColumnSkeleton />}
            {column.key === "action" && <ActionColumnSkeleton />}
          </td>
        ))
      }
    />
  );
};
