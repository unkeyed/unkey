"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { columns } from "./columns/sentinel-logs";
import { useDeploymentSentinelLogsQuery } from "./hooks/use-deployment-sentinel-logs-query";
import { getRowClassName } from "./utils/get-sentinel-logs-row-class";

export const DeploymentSentinelLogsTable = () => {
  const { logs, isLoading } = useDeploymentSentinelLogsQuery();
  return (
    <VirtualTable
      data={logs}
      isLoading={isLoading}
      columns={columns}
      keyExtractor={(log) => log.request_id}
      rowClassName={(log) => getRowClassName(log)}
      fixedHeight={600}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Deployment Logs</Empty.Title>
            <Empty.Description className="text-left">
              No logs found for this deployment. Logs appear here when your deployment receives
              requests.
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
        </div>
      }
    />
  );
};
