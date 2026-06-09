"use client";

import { StreamingTable } from "@/components/streaming-table";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
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
        <Empty className="w-100 flex items-start">
          <Empty.Icon className="w-auto" />
          <Empty.Title>Container Logs</Empty.Title>
          <Empty.Description className="text-left">
            No runtime logs found for this deployment. Container logs will appear here once the
            deployment starts running.
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
      }
    />
  );
};
