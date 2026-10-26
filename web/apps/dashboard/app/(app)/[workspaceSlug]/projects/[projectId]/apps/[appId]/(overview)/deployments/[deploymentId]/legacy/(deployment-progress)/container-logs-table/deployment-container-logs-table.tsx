"use client";

import { StreamingTable } from "@/components/streaming-table";
import { IconBookBookmarkOutline18, IconSquareBulletListOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { type ContainerLogRow, containerLogColumns } from "./columns";
import { getContainerLogRowClass } from "./get-row-class";
import {
  MessageColumnSkeleton,
  RegionColumnSkeleton,
  SeverityColumnSkeleton,
  TimeColumnSkeleton,
} from "./skeletons";

type Props = {
  logs: ContainerLogRow[];
  isLoading: boolean;
};

export const DeploymentContainerLogsTable = ({ logs, isLoading }: Props) => {
  return (
    <StreamingTable
      data={logs}
      columns={containerLogColumns}
      keyExtractor={(log) => `${log.time}-${log.instance_id}-${log.region}-${log.severity}`}
      rowClassName={getContainerLogRowClass}
      renderSkeletonCell={(col) => {
        switch (col.key) {
          case "log":
            return <TimeColumnSkeleton />;
          case "severity":
            return <SeverityColumnSkeleton />;
          case "region":
            return <RegionColumnSkeleton />;
          case "message":
            return <MessageColumnSkeleton />;
          default:
            return null;
        }
      }}
      isLoading={isLoading}
      fixedHeight={500}
      emptyState={
        <EmptyState frame="none">
          <EmptyStateIcon>
            <IconSquareBulletListOutline18 />
          </EmptyStateIcon>
          <EmptyStateHeader>
            <EmptyStateTitle>Container Logs</EmptyStateTitle>
            <EmptyStateDescription>
              No runtime logs found for this deployment. Container logs will appear here once the
              deployment starts running.
            </EmptyStateDescription>
          </EmptyStateHeader>
          <EmptyStateActions>
            <a
              href="https://www.unkey.com/docs/introduction"
              target="_blank"
              rel="noopener noreferrer"
            >
              <Button variant="outline" size="md">
                <IconBookBookmarkOutline18 />
                Documentation
              </Button>
            </a>
          </EmptyStateActions>
        </EmptyState>
      }
    />
  );
};
