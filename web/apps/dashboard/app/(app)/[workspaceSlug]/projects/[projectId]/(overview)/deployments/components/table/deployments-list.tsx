"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment, Environment } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { formatCpuParts, formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { Bolt, BookBookmark, CodeBranch, Connections3, ScanCode } from "@unkey/icons";
import { Button, Empty, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { useCallback, useMemo } from "react";
import { DeploymentStatusBadge } from "../../../../components/deployment-status-badge";
import { Avatar } from "../../../../components/git-avatar";
import { StatusIndicator } from "../../../../components/status-indicator";
import { useProjectData } from "../../../data-provider";
import { useDeployments } from "../../hooks/use-deployments";
import { DomainList } from "./components/domain_list";
import { EnvStatusBadge } from "./components/env-status-badge";
import {
  ActionColumnSkeleton,
  CreatedAtColumnSkeleton,
  DeploymentIdColumnSkeleton,
  EnvColumnSkeleton,
  InstancesColumnSkeleton,
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
  const { deployments } = useDeployments();
  const { project } = useProjectData();
  const currentDeploymentId = project?.currentDeploymentId;

  const workspace = useWorkspaceNavigation();
  const router = useRouter();

  const getDeploymentHref = useCallback(
    (deploymentId: string) =>
      `/${workspace.slug}/projects/${project?.id}/deployments/${deploymentId}`,
    [workspace.slug, project?.id],
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
        width: "12%",
        headerClassName: "pl-[18px]",
        render: ({ deployment, environment }) => {
          const isCurrent = currentDeploymentId === deployment.id;
          const iconContainer = <StatusIndicator withSignal={isCurrent} />;
          return (
            <div className="flex flex-col items-start px-4.5 py-1.5">
              <div className="flex gap-3 items-center w-full">
                <div className="shrink-0">{iconContainer}</div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <div
                      className={cn(
                        "font-normal font-mono truncate leading-5 text-[13px]",
                        "text-accent-12",
                      )}
                    >
                      {shortenId(deployment.id)}
                    </div>
                    {isCurrent ? (
                      <div className="shrink-0">
                        {project?.isRolledBack ? (
                          <EnvStatusBadge variant="rolledBack" text="Rolled Back" />
                        ) : (
                          <EnvStatusBadge variant="current" text="Current" />
                        )}
                      </div>
                    ) : null}
                  </div>
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
        width: "10%",
        render: ({ deployment }) => <DeploymentStatusBadge status={deployment.status} />,
      },
      {
        key: "domains",
        header: "Domains",
        width: "10%",
        render: ({ deployment }) => {
          return (
            <div className="flex items-center min-h-[52px]">
              <DomainList deploymentId={deployment.id} status={deployment.status} />
            </div>
          );
        },
      },
      {
        key: "details",
        header: "",
        width: "15%",
        render: ({ deployment }: { deployment: Deployment }) => {
          const hideResources =
            deployment.status === "failed" ||
            deployment.status === "skipped" ||
            deployment.status === "stopped";
          const cpu = hideResources ? null : formatCpuParts(deployment.cpuMillicores);
          const mem = hideResources ? null : formatMemoryParts(deployment.memoryMib);
          return (
            <div className="flex items-center gap-7">
              <div className="hidden 2xl:flex items-center w-[80px]">
                {hideResources ? (
                  <span className="text-gray-9">—</span>
                ) : (
                  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md text-grayA-11 w-fit h-[22px]">
                    <Connections3 className="text-gray-12" iconSize="sm-regular" />
                    <span className="font-semibold text-grayA-12 tabular-nums">
                      {deployment.instances.length}
                    </span>
                    <span>VMs</span>
                  </div>
                )}
              </div>
              <div className="hidden 2xl:flex gap-1.5 w-[180px]">
                {hideResources || !cpu || !mem ? (
                  <span className="text-gray-9">—</span>
                ) : (
                  <>
                    <div className="bg-grayA-3 font-mono text-xs items-center flex gap-1.5 p-1.5 rounded-md text-grayA-11 w-fit h-[22px]">
                      <Bolt className="text-gray-12" iconSize="sm-regular" />
                      <span className="font-semibold text-grayA-12">{cpu.value}</span>
                      <span>{cpu.unit}</span>
                    </div>
                    <div className="bg-grayA-3 font-mono text-xs items-center flex gap-1.5 p-1.5 rounded-md text-grayA-11 w-fit h-[22px]">
                      <ScanCode className="text-gray-12" iconSize="sm-regular" />
                      <span className="font-semibold text-grayA-12">{mem.value}</span>
                      <span>{mem.unit}</span>
                    </div>
                  </>
                )}
              </div>
              <div className="flex gap-2 items-center max-w-50">
                <div className="size-5 rounded flex items-center justify-center border border-grayA-3 bg-grayA-3">
                  <CodeBranch iconSize="sm-regular" className="text-gray-12" />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 font-mono text-[13px]">
                    <span className="truncate text-accent-12 max-w-25" title={deployment.gitBranch}>
                      {deployment.gitBranch}
                    </span>
                    <span className="text-gray-6 shrink-0">·</span>
                    <span className="text-gray-9 shrink-0">
                      {deployment.gitCommitSha?.slice(0, 7)}
                    </span>
                  </div>
                  {deployment.gitCommitMessage ? (
                    <div
                      className="truncate text-xs text-gray-9 w-50 pr-5"
                      title={deployment.gitCommitMessage}
                    >
                      {deployment.gitCommitMessage}
                    </div>
                  ) : null}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Avatar
                  src={deployment.gitCommitAuthorAvatarUrl}
                  alt={deployment.gitCommitAuthorHandle ?? "Author"}
                />
                <span className="font-medium text-grayA-12 text-xs">
                  {deployment.gitCommitAuthorHandle || "—"}
                </span>
              </div>
            </div>
          );
        },
      },
      {
        key: "created_at" as const,
        header: "Created",
        width: "8%",
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
        key: "action",
        header: "",
        width: "4%",
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
                environment={environment}
              />
            </div>
          );
        },
      },
    ];
  }, [project]);

  return (
    <VirtualTable
      data={deployments.data}
      isLoading={deployments.isLoading}
      columns={columns}
      onRowClick={(item) => {
        if (item) {
          router.push(getDeploymentHref(item.deployment.id));
        }
      }}
      onRowMouseEnter={(item) => {
        router.prefetch(getDeploymentHref(item.deployment.id));
      }}
      keyExtractor={(deployment) => deployment.id}
      rowClassName={(deployment) =>
        getRowClassName(deployment, currentDeploymentId ?? null, project?.isRolledBack ?? false)
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
        tableLayout: "auto",
      }}
      renderSkeletonRow={({ columns, rowHeight }) =>
        columns.map((column) => (
          <td
            key={column.key}
            className={cn("text-xs align-middle whitespace-nowrap", column.cellClassName)}
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "deployment_id" && <DeploymentIdColumnSkeleton />}
            {column.key === "env" && <EnvColumnSkeleton />}
            {column.key === "status" && <StatusColumnSkeleton />}
            {column.key === "details" && <InstancesColumnSkeleton />}
            {column.key === "domains" && (
              <div className="flex items-center min-h-[52px]">
                <div className="h-4 bg-grayA-3 rounded w-32 animate-pulse" />
              </div>
            )}
            {column.key === "created_at" && <CreatedAtColumnSkeleton />}
            {column.key === "action" && <ActionColumnSkeleton />}
          </td>
        ))
      }
    />
  );
};
