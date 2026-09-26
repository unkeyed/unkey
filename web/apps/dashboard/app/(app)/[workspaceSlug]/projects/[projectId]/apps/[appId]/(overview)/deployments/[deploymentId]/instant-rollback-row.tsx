"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import type { Deployment } from "@/lib/collections";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { previousRollbackTarget } from "@/lib/collections/deploy/rollback";
import { IconArrowDottedRotateAnticlockwiseOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useState } from "react";
import { useProjectData } from "../../data-provider";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { useDeployment } from "./layout-provider";

const RollbackDialog = dynamic(
  () =>
    import("../components/table/components/actions/rollback-dialog").then((m) => m.RollbackDialog),
  { ssr: false },
);

export function InstantRollbackRow() {
  const { deployment } = useDeployment();
  const { deployments, environments } = useProjectData();
  const { app, isRolledBack } = useAppCurrentDeployment();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const [rollbackOpen, setRollbackOpen] = useState(false);
  const [dialogTarget, setDialogTarget] = useState<Deployment>();

  const environment = environments.find((e) => e.id === deployment.environmentId);
  const isLiveProduction =
    environment?.kind === ENVIRONMENT_KIND.production &&
    app?.currentDeploymentId === deployment.id &&
    !isRolledBack;
  const target = isLiveProduction ? previousRollbackTarget(deployments, deployment) : undefined;

  const openRollback = (next: Deployment) => {
    if (gated) {
      openPaywall();
      return;
    }
    setDialogTarget(next);
    setRollbackOpen(true);
  };

  return (
    <>
      {target && (
        <div className="flex flex-wrap items-start gap-3 border-t px-4 py-3.5 sm:flex-nowrap">
          <div className="hidden size-7 shrink-0 items-center justify-center self-center rounded-full border bg-grayA-2 sm:flex">
            <IconArrowDottedRotateAnticlockwiseOutline18 className="size-3.5 text-gray-12" />
          </div>
          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="text-sm font-medium leading-5 text-gray-12">
              Need to revert these changes?
            </span>
            <p className="text-sm leading-5 text-gray-11">
              Your previous deployment is still running at full scale. Switch traffic back instantly
              without cold starts.
            </p>
          </div>
          <div className="w-full shrink-0 self-center sm:w-auto">
            <Button
              variant="outline"
              size="sm"
              onClick={() => openRollback(target)}
              className="px-3"
            >
              Instant Rollback
            </Button>
          </div>
        </div>
      )}
      {rollbackOpen && dialogTarget && (
        <RollbackDialog
          isOpen={rollbackOpen}
          onClose={() => setRollbackOpen(false)}
          targetDeployment={dialogTarget}
          currentDeployment={deployment}
        />
      )}
      {planGate}
    </>
  );
}
