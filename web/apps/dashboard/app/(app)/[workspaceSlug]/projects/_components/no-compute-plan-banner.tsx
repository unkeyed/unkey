"use client";

import { TriangleWarning2 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { DeployPlanGateDialog } from "./deploy-plan-gate-dialog";
import { useDeployGate } from "./hooks/use-deploy-gate";

/**
 * Warning banner shown inside a project when the workspace has no active
 * Compute plan. Lives on the project Apps page (where "Create app" is) rather
 * than the workspace projects list, since app creation is what the missing
 * plan actually pauses. Renders nothing while entitled or loading (fail-open,
 * like the rest of the gate UX).
 */
export function NoComputePlanBanner() {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);

  if (!gated) {
    return null;
  }

  return (
    <>
      <div className="mb-4 flex items-center justify-between gap-4 rounded-lg border border-warningA-6 bg-warningA-2 px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <TriangleWarning2 iconSize="md-regular" className="shrink-0 text-warning-11" />
          <p className="truncate text-[13px] text-gray-11">
            No active Compute plan. Creating apps and deploying are paused.
          </p>
        </div>
        <Button
          variant="outline"
          size="md"
          className="bg-background"
          onClick={() => setIsPlanOpen(true)}
        >
          Choose a plan
        </Button>
      </div>
      <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="banner" />
    </>
  );
}
