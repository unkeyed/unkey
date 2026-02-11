"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useState } from "react";
import { BuildStepLogsExpanded } from "./build-step-logs-expanded";
import { buildStepsColumns } from "./columns/build-steps";
import { useDeploymentBuildStepsQuery } from "./hooks/use-deployment-build-steps-query";
import { getBuildStepRowClass } from "./utils/get-build-step-row-class";

export const DeploymentBuildStepsTable = () => {
  const { steps, isLoading } = useDeploymentBuildStepsQuery();
  const [expandedIds, setExpandedIds] = useState<Set<string | number>>(new Set());

  // Enrich steps with expansion state for chevron rendering
  const enrichedSteps = steps.map((step) => ({
    ...step,
    _isExpanded: expandedIds.has(step.step_id),
  }));

  return (
    <VirtualTable
      data={enrichedSteps}
      isLoading={isLoading}
      columns={buildStepsColumns}
      keyExtractor={(step) => step.step_id}
      rowClassName={(step) => getBuildStepRowClass(step)}
      fixedHeight={600}
      expandedIds={expandedIds}
      onExpandedChange={setExpandedIds}
      isExpandable={(step) => step.has_logs}
      renderExpanded={(step) => <BuildStepLogsExpanded step={step} />}
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
