"use client";
import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useIsMobile } from "@/hooks/use-mobile";
import { type Deployment, type Environment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { eq, gt, gte, lte, or, useLiveQuery } from "@tanstack/react-db";
import { BookBookmark, Cloud, CodeBranch, Cube } from "@unkey/icons";
import { Button, Empty, TimestampInfo } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import ms from "ms";
import dynamic from "next/dynamic";
import { useMemo, useState } from "react";
import type { DeploymentListFilterField } from "../../filters.schema";
import { useFilters } from "../../hooks/use-filters";
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
  const { filters } = useFilters();

  const deployments = useLiveQuery(
    (q) => {
      // Query filtered environments
      // further down below we use this to rightJoin with deployments to filter deployments by environment
      let environments = q.from({ environment: collection.environments });

      for (const filter of filters) {
        if (filter.field === "environment") {
          environments = environments.where(({ environment }) =>
            eq(environment.slug, filter.value),
          );
        }
      }

      let query = q
        .from({ deployment: collection.deployments })

        .where(({ deployment }) => eq(deployment.projectId, projectId));

      // add additional where clauses based on filters.
      // All of these are a locical AND

      const groupedFilters = filters.reduce(
        (acc, f) => {
          if (!acc[f.field]) {
            acc[f.field] = [];
          }
          acc[f.field].push(f.value);
          return acc;
        },
        {} as Record<DeploymentListFilterField, (string | number)[]>,
      );
      for (const [field, values] of Object.entries(groupedFilters)) {
        // this is kind of dumb, but `or`s type doesn't allow spreaded args without
        // specifying the first two
        const [v1, v2, ...rest] = values;
        const f = field as DeploymentListFilterField; // I want some typesafety
        switch (f) {
          case "status":
            query = query.where(({ deployment }) =>
              or(
                eq(deployment.status, v1),
                eq(deployment.status, v2),
                ...rest.map((value) => eq(deployment.status, value)),
              ),
            );
            break;
          case "branch":
            query = query.where(({ deployment }) =>
              or(
                eq(deployment.gitBranch, v1),
                eq(deployment.gitBranch, v2),
                ...rest.map((value) => eq(deployment.gitBranch, value)),
              ),
            );
            break;
          case "environment":
            // We already filtered
            break;
          case "since":
            query = query.where(({ deployment }) =>
              gt(deployment.createdAt, Date.now() - ms(values.at(0) as string)),
            );

            break;
          case "startTime":
            query = query.where(({ deployment }) => gte(deployment.createdAt, values.at(0)));
            break;
          case "endTime":
            query = query.where(({ deployment }) => lte(deployment.createdAt, values.at(0)));
            break;
          default:
            break;
        }
      }

      return query
        .rightJoin({ environment: environments }, ({ environment, deployment }) =>
          eq(environment.id, deployment.environmentId),
        )
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(100);
    },
    [projectId, filters],
  );

  const [selectedDeployment, setSelectedDeployment] = useState<{
    deployment: Deployment;
    environment?: Environment;
  } | null>(null);
  const isCompactView = useIsMobile({ breakpoint: COMPACT_BREAKPOINT });

  const columns: Column<{ deployment: Deployment; environment?: Environment }>[] = useMemo(() => {
    return [
      {
        key: "deployment_id",
        header: "Deployment ID",
        width: "20%",
        headerClassName: "pl-[18px]",
        render: ({ deployment, environment }) => {
          const isSelected = deployment.id === selectedDeployment?.deployment.id;
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
                    {environment?.slug === "production" ? (
                      <EnvStatusBadge variant="current" text="Current" />
                    ) : null}
                  </div>
                  <div className={cn("font-normal font-mono truncate text-xs mt-1", "text-gray-9")}>
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
        render: ({ deployment }) => {
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
              render: ({ deployment }: { deployment: Deployment }) => {
                return (
                  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
                    <Cube className="text-gray-12" size="sm-regular" />
                    <div className="flex gap-0.5">
                      <span className="font-semibold text-grayA-12 tabular-nums">
                        {deployment.runtimeConfig.regions.reduce(
                          (acc, region) => acc + region.vmCount,
                          0,
                        )}
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
              render: ({ deployment }: { deployment: Deployment }) => {
                return (
                  <div className="bg-grayA-3 font-mono text-xs items-center flex gap-2 p-1.5 rounded-md relative text-grayA-11 w-fit">
                    <Cube className="text-gray-12" size="sm-regular" />
                    <div className="flex gap-1">
                      <div className="flex gap-0.5">
                        <span className="font-semibold text-grayA-12 tabular-nums">
                          {deployment.runtimeConfig.cpus}
                        </span>
                        <span>CPU</span>
                      </div>
                      <span> / </span>
                      <div className="flex gap-0.5">
                        <span className="font-semibold text-grayA-12 tabular-nums">
                          {deployment.runtimeConfig.memory}
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
        width: "20%",
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
              render: ({ deployment }: { deployment: Deployment }) => {
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
        render: ({ deployment }: { deployment: Deployment }) => {
          return <DeploymentListTableActions deployment={deployment} />;
        },
      },
    ];
  }, [selectedDeployment, isCompactView]);

  return (
    <VirtualTable
      data={deployments.data}
      isLoading={deployments.isLoading}
      columns={columns}
      onRowClick={setSelectedDeployment}
      selectedItem={selectedDeployment}
      keyExtractor={(deployment) => deployment.id}
      rowClassName={(deployment) => getRowClassName(deployment, selectedDeployment?.deployment.id)}
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
