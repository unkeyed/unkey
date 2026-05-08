"use client";

import { StreamingTable } from "@/components/streaming-table";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
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
  isLoading: boolean;
  fixedHeight?: number;
  focusStepId?: string | null;
};

export const DeploymentBuildStepsTable: React.FC<Props> = ({
  steps,
  isLoading,
  fixedHeight = 500,
  focusStepId,
}) => {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!focusStepId) {
      return;
    }
    const stepId = focusStepId.split("#")[0];
    setExpandedIds((prev) => (prev.has(stepId) ? prev : new Set(prev).add(stepId)));
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        containerRef.current
          ?.querySelector<HTMLElement>(`[data-row-id="${stepId}"]`)
          ?.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    });
  }, [focusStepId]);

  const toggleExpand = (step: BuildStepRow) => {
    if (!step.has_logs) {
      return;
    }
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(step.step_id)) {
        next.delete(step.step_id);
      } else {
        next.add(step.step_id);
      }
      return next;
    });
  };

  const enrichedSteps = steps.map((step) => ({
    ...step,
    _isExpanded: expandedIds.has(step.step_id),
  }));

  return (
    <div ref={containerRef}>
      <StreamingTable
        data={enrichedSteps}
        columns={buildStepsColumns}
        keyExtractor={(step) => step.step_id}
        rowClassName={getBuildStepRowClass}
        onRowClick={toggleExpand}
        renderExpanded={(step) =>
          expandedIds.has(step.step_id) ? <BuildStepLogsExpanded step={step} /> : null
        }
        renderSkeletonCell={(col) => {
          switch (col.key) {
            case "started_at":
              return <StartedAtColumnSkeleton />;
            case "status":
              return <StatusColumnSkeleton />;
            case "name":
              return <NameColumnSkeleton />;
            case "duration":
              return <DurationColumnSkeleton />;
            default:
              return null;
          }
        }}
        isLoading={isLoading}
        fixedHeight={fixedHeight}
        emptyState={
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
        }
      />
    </div>
  );
};
