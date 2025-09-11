"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useIsMobile } from "@/hooks/use-mobile";
import { shortenId } from "@/lib/shorten-id";
import { BookBookmark, Cloud, CodeBranch, Cube } from "@unkey/icons";
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
  SizeColumnSkeleton,
  SourceColumnSkeleton,
  StatusColumnSkeleton,
} from "./components/skeletons";
import { getRowClassName } from "./utils/get-row-class";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { collection, type Environment, type Deployment } from "@/lib/collections";

const DeploymentListTableActions = dynamic(
  () =>
    import("./components/actions/deployment-list-table-action.popover.constants").then(
      (mod) => mod.DeploymentListTableActions,
    ),
  {
    loading: () => <ActionColumnSkeleton />,
    ssr: false,
  },
);

const COMPACT_BREAKPOINT = 1200;

type Props = {
  projectId: string;
};

export const DeploymentsList = ({ projectId }: Props) => {
  const deployments = useLiveQuery((q) =>
    q
      .from({ deployment: collection.deployments })
      .where(({ deployment }) => eq(deployment.projectId, projectId))
      .join({ environment: collection.environments }, ({ environment, deployment }) => eq(environment.id, deployment.environmentId))
      .orderBy(({ deployment }) => deployment.createdAt, "desc")
      .limit(100),
  );


  const [selectedDeployment, setSelectedDeployment] = useState<Deployment & { environment: Environment } | null>(null);
  const isCompactView = useIsMobile({ breakpoint: COMPACT_BREAKPOINT });

  const columns: Column<Deployment & { environment: Environment }>[] = useMemo(() => {
    return [
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
                    {deployment.environment.slug === "production" ?

                      <EnvStatusBadge variant="current" text="Current" />
                      : null}
                  </div>
                  <div className={cn("font-normal font-mono truncate text-xs mt-1", "text-gray-9")}>
                    {deployment.gitCommitMessage?.slice(0, 64) ?? "—"}
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
        width: "15%",
        render: (deployment) => {
          return (
            <div className="bg-grayA-3 text-xs items-center flex gap-2 p-1.5 rounded-md relative w-fit capitalize">
              {deployment.environment.slug}
            </div>
          );
        },
      },
      {
        key: "status",
        header: "Status",
        width: "12%",
        render: (deployment) => {
          return <DeploymentStatusBadge status={deployment.status} />;
        },
      },
      ...(isCompactView
        ? []
        : [
          {
            key: "instances" as const,
            header: "Instances",
            width: "10%",
            render: (_deployment: Deployment) => {
              return (
                <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
                  <Cube className="text-gray-12" size="sm-regular" />
                  <div className="flex gap-0.5">
                    <span className="font-semibold text-grayA-12 tabular-nums">
                      TODO
                    </span>
                    <span>VMs</span>
                  </div>
                </div>
              );
            },
          },
          {
            key: "size" as const,
            header: "Size",
            width: "10%",
            render: (_deployment: Deployment) => {
              return (
                <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
                  <Cube className="text-gray-12" size="sm-regular" />
                  <div className="flex gap-1">
                    <div className="flex gap-0.5">
                      <span className="font-semibold text-grayA-12 tabular-nums">2</span>
                      <span>CPU</span>
                    </div>
                    <span> / </span>
                    <div className="flex gap-0.5">
                      <span className="font-semibold text-grayA-12 tabular-nums">
                        TODO
                      </span>
                      <span>MB</span>
                    </div>
                  </div>
                </div>
              );
            },
          },
        ]),
      {
        key: "source",
        header: "Source",
        width: "10%",
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
              <CodeBranch size="sm-regular" className="text-gray-12" />
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
                      {deployment.gitBranch}
                    </div>
                  </div>
                  <div className={cn("font-normal font-mono truncate text-xs mt-1", "text-gray-9")}>
                    {deployment.gitCommitSha}
                  </div>
                </div>
              </div>
            </div>
          );
        },
      },
      ...(isCompactView
        ? [
          {
            key: "author_created" as const,
            header: "Author / Created",
            width: "20%",
            render: (deployment: Deployment) => {
              return (
                <div className="flex flex-col items-start pr-[18px] py-1.5">
                  <div className="flex gap-5 items-center w-full">
                    <img
                      src={deployment.gitCommitAuthorAvatarUrl ?? ""}
                      alt="Author"
                      className="rounded-full size-5"
                    />
                    <div className="w-[200px]">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-grayA-12 text-xs">
                          {deployment.gitCommitAuthorUsername}
                        </span>
                      </div>
                      <div className={cn("font-mono text-xs mt-1", "text-gray-9")}>
                        <TimestampInfo
                          value={deployment.createdAt}
                          className="font-mono text-xs text-gray-9"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              );
            },
          },
        ]
        : [
          {
            key: "created_at" as const,
            header: "Created at",
            width: "10%",
            render: (deployment: Deployment) => {
              return (
                <TimestampInfo
                  value={deployment.createdAt}
                  className="font-mono group-hover:underline decoration-dotted"
                />
              );
            },
          },
          {
            key: "author" as const,
            header: "Author",
            width: "10%",
            render: (deployment: Deployment) => {
              return (
                <div className="flex items-center gap-2">
                  <img
                    src={deployment.gitCommitAuthorAvatarUrl ?? ""}
                    alt="Author"
                    className="rounded-full size-5"
                  />
                  <span className="font-medium text-grayA-12 text-xs">
                    {deployment.gitCommitAuthorName}
                  </span>
                </div>
              );
            },
          },
        ]),
      {
        key: "action",
        header: "",
        width: "auto",
        render: (deployment) => {
          return <DeploymentListTableActions deployment={deployment} />;
        },
      },
    ];
  }, [selectedDeployment?.id, isCompactView]);

  return (
    <VirtualTable
      data={Object.values(deployments.data).map((e) => ({
        ...e.deployment,
        environment: e.environment!,
      } satisfies Deployment & { environment: Environment }))}
      isLoading={deployments.isLoading}
      columns={columns}
      onRowClick={setSelectedDeployment}
      selectedItem={selectedDeployment}
      keyExtractor={(deployment) => deployment.id}
      rowClassName={(deployment) => getRowClassName(deployment, selectedDeployment?.id)}
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
            {column.key === "size" && <SizeColumnSkeleton />}
            {column.key === "source" && <SourceColumnSkeleton />}
            {column.key === "created_at" && <CreatedAtColumnSkeleton />}
            {column.key === "author" && <AuthorColumnSkeleton />}
            {column.key === "author_created" && (
              <div className="flex flex-col gap-1">
                <AuthorColumnSkeleton />
                <CreatedAtColumnSkeleton />
              </div>
            )}
            {column.key === "action" && <ActionColumnSkeleton />}
          </td>
        ))
      }
    />
  );
};
