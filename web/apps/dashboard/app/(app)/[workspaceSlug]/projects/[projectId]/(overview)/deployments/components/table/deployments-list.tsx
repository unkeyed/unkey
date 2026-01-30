"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { BookBookmark, CodeBranch, Cube } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { Button, Empty, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useMemo, useState } from "react";
import { Avatar } from "../../../../components/git-avatar";
import { StatusIndicator } from "../../../../components/status-indicator";
import { useDeployments } from "../../hooks/use-deployments";
import { DeploymentStatusBadge } from "./components/deployment-status-badge";
import { DomainList } from "./components/domain_list";
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

export const DeploymentsList = () => {
  const [selectedDeployment, setSelectedDeployment] = useState<{
    deployment: Deployment;
    environment?: Environment;
  } | null>(null);
  const { liveDeployment, deployments, project } = useDeployments();

  const selectedDeploymentId = selectedDeployment?.deployment.id;

  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [navigatingId, setNavigatingId] = useState<string | null>(null);

  const handleLinkClick = useCallback(
    (e: React.MouseEvent, deploymentId: string) => {
      e.preventDefault();
      setNavigatingId(deploymentId);
      router.push(`/${workspace.slug}/projects/${project?.id}/deployments/${deploymentId}`);
    },
    [router, workspace.slug, project?.id],
  );

  const columns: Column<{
    deployment: Deployment;
    environment?: Environment;
    // biome-ignore lint/correctness/useExhaustiveDependencies: its okay
  }>[] = useMemo(() => {
    return [
      {
        key: "deployment_id",
        header: "Deployment ID",
        width: "15%",
        headerClassName: "pl-[18px]",
        render: ({ deployment, environment }) => {
          const isLive = liveDeployment?.id === deployment.id;
          const iconContainer = <StatusIndicator withSignal={isLive} />;
          return (
            <div className="flex flex-col items-start px-[18px] py-1.5">
              <div className="flex gap-3 items-center w-full">
                {navigatingId === deployment.id ? (
                  <div className="relative flex-shrink-0">
                    <div className="size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3">
                      <div className="size-[12px] items-center justify-center flex">
                        <Loading size={18} />
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="flex-shrink-0">{iconContainer}</div>
                )}
                <div className="min-w-0 flex-1">
                  <Link
                    title={`View details for ${deployment.id}`}
                    className="font-mono group-hover:underline decoration-dotted"
                    href={`/${workspace.slug}/projects/${project?.id}/deployments/${deployment.id}`}
                    onClick={(e) => handleLinkClick(e, deployment.id)}
                  >
                    <div className="flex items-center gap-2">
                      <div
                        className={cn(
                          "font-normal font-mono truncate leading-5 text-[13px]",
                          "text-accent-12",
                        )}
                      >
                        {shortenId(deployment.id)}
                      </div>
                      {isLive ? (
                        <div className="flex-shrink-0">
                          {project?.isRolledBack ? (
                            <EnvStatusBadge variant="rolledBack" text="Rolled Back" />
                          ) : (
                            <EnvStatusBadge variant="live" text="Live" />
                          )}
                        </div>
                      ) : null}
                    </div>
                  </Link>
                  <div
                    className={cn(
                      "font-normal font-mono truncate text-xs mt-1 capitalize",
                      "text-gray-9",
                    )}
                  >
                    {environment?.slug}
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
        width: "12%",
        render: ({ deployment }) => <DeploymentStatusBadge status={deployment.status} />,
      },
      {
        key: "domains",
        header: "Domains",
        width: "20%",
        render: ({ deployment }) => (
          <div className="flex items-center min-h-[52px]">
            <DomainList
              key={`${deployment.id}-${liveDeployment}-${project?.isRolledBack}-${project?.latestDeploymentId}`}
              deploymentId={deployment.id}
            />
          </div>
        ),
      },
      {
        key: "instances" as const,
        header: "Instances",
        width: "10%",
        render: ({ deployment }: { deployment: Deployment }) => {
          return (
            <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
              <Cube className="text-gray-12" iconSize="sm-regular" />
              <div className="flex gap-0.5">
                <span className="font-semibold text-grayA-12 tabular-nums">
                  {deployment.instances.length}
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
        width: "15%",
        render: ({ deployment }: { deployment: Deployment }) => {
          return (
            <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
              <Cube className="text-gray-12" iconSize="sm-regular" />
              <div className="flex gap-1">
                <div className="flex gap-0.5">
                  <span className="font-semibold text-grayA-12 tabular-nums">
                    {deployment.cpuMillicores / 1024}
                  </span>
                  <span>vCPU</span>
                </div>
                <span> / </span>
                <div className="flex gap-0.5">
                  <span className="font-semibold text-grayA-12 tabular-nums">
                    {deployment.memoryMib}
                  </span>
                  <span>MiB</span>
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "source",
        header: "Source",
        width: "13%",
        headerClassName: "pl-[18px]",
        render: ({ deployment }) => {
          const isSelected = deployment.id === selectedDeployment?.deployment.id;
          const iconContainer = (
            <div
              className={cn(
                "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
                "bg-grayA-3",
                isSelected && "bg-grayA-5",
              )}
            >
              <CodeBranch iconSize="sm-regular" className="text-gray-12" />
            </div>
          );
          return (
            <div className="flex flex-col items-start px-[18px] py-1.5">
              <div className="flex gap-3 items-center w-full">
                <div className="flex-shrink-0">{iconContainer}</div>
                <div className="min-w-0 flex-1">
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
                    {deployment.gitCommitSha?.slice(0, 7)}
                  </div>
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "created_at" as const,
        header: "Created",
        width: "10%",
        render: ({ deployment }: { deployment: Deployment }) => {
          return (
            <TimestampInfo
              value={deployment.createdAt}
              displayType="relative"
              className="font-mono group-hover:underline decoration-dotted"
            />
          );
        },
      },
      {
        key: "author" as const,
        header: "Author",
        width: "10%",
        render: ({ deployment }: { deployment: Deployment }) => {
          return (
            <div className="flex items-center gap-2">
              <Avatar
                src={deployment.gitCommitAuthorAvatarUrl}
                alt={deployment.gitCommitAuthorHandle ?? "Author"}
              />
              <span className="font-medium text-grayA-12 text-xs">
                {deployment.gitCommitAuthorHandle || "—"}
              </span>
            </div>
          );
        },
      },
      {
        key: "action",
        header: "",
        width: "5%",
        render: ({
          deployment,
          environment,
        }: {
          deployment: Deployment;
          environment?: Environment;
        }) => {
          return (
            <div className="pl-5">
              <DeploymentListTableActions
                selectedDeployment={deployment}
                liveDeployment={liveDeployment}
                environment={environment}
              />
            </div>
          );
        },
      },
    ];
  }, [selectedDeploymentId, liveDeployment, project]);

  return (
    <VirtualTable
      data={deployments.data}
      isLoading={deployments.isLoading}
      columns={columns}
      onRowClick={setSelectedDeployment}
      selectedItem={selectedDeployment}
      keyExtractor={(deployment) => deployment.id}
      rowClassName={(deployment) =>
        getRowClassName(
          deployment,
          selectedDeployment?.deployment.id ?? null,
          liveDeployment?.id ?? null,
          project?.isRolledBack ?? false,
        )
      }
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
            {column.key === "domains" && (
              <div className="flex items-center min-h-[52px]">
                <div className="h-4 bg-grayA-3 rounded w-32 animate-pulse" />
              </div>
            )}
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
