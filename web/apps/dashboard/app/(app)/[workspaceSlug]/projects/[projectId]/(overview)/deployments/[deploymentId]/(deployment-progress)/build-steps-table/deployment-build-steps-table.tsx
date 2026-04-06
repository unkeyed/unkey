"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import { cn } from "@/lib/utils";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { BuildStepLogsExpanded } from "./build-step-logs-expanded";
import { type BuildStepRow, buildStepsColumns } from "./columns";
import { getBuildStepRowClass } from "./get-row-class";
import {
  DurationColumnSkeleton,
  NameColumnSkeleton,
  StartedAtColumnSkeleton,
  StatusColumnSkeleton,
} from "./skeletons";

type Props = {
  steps: BuildStepRow[];
  fixedHeight?: number;
};

export const DeploymentBuildStepsTable: React.FC<Props> = ({ steps, fixedHeight = 500 }) => {
  const [expandedIds, setExpandedIds] = useState<Set<string | number>>(new Set());

  const enrichedSteps = steps.map((step) => ({
    ...step,
    _isExpanded: expandedIds.has(step.step_id),
  }));

  return (
    <VirtualTable
      data={enrichedSteps}
      isLoading={steps.length === 0}
      columns={buildStepsColumns}
      renderSkeletonRow={({ columns, rowHeight }) =>
        columns.map((column, idx) => (
          <td
            key={column.key}
            className={cn(
              "text-xs align-middle whitespace-nowrap",
              idx === 0 ? "pl-4.5" : "",
              column.cellClassName,
            )}
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "started_at" && <StartedAtColumnSkeleton />}
            {column.key === "status" && <StatusColumnSkeleton />}
            {column.key === "name" && <NameColumnSkeleton />}
            {column.key === "duration" && <DurationColumnSkeleton />}
          </td>
        ))
      }
      keyExtractor={(step) => step.step_id}
      rowClassName={(step) => getBuildStepRowClass(step)}
      expandedIds={expandedIds}
      onExpandedChange={setExpandedIds}
      fixedHeight={fixedHeight}
      autoScrollToBottom
      isExpandable={(step) => step.has_logs}
      renderExpanded={(step) => <BuildStepLogsExpanded step={step} />}
      config={{
        containerPadding: "px-0 py-0",
        className: "bg-transparent",
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>Build Steps</Empty.Title>
            <Empty.Description className="text-left">
              No build steps found for this deployment. Build steps will appear here once the
              deployment starts building.
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
