"use client";

import { useCallback, useState } from "react";
import { DeployPlanGateDialog } from "../deploy-plan-gate-dialog";
import { useDeployGate } from "./use-deploy-gate";

type DeployActionGate = {
  /** True when deployBilling is on and the workspace has no Compute plan. */
  gated: boolean;
  /** Opens the Compute-plan paywall. */
  openPaywall: () => void;
  /** Render this once in the component tree to host the paywall dialog. */
  planGate: React.ReactNode;
};

/**
 * Shared Compute paywall for deploy actions inside apps (create, redeploy,
 * promote, rollback, wake, ...). When the workspace is gated, callers route the
 * action's trigger to `openPaywall` instead of performing the deployment, and
 * render `planGate` somewhere in their tree. ctrl-api is the authoritative
 * gate; this is UX only and fails open (see useDeployGate).
 */
export function useDeployActionGate(): DeployActionGate {
  const { gated } = useDeployGate();
  const [isPlanOpen, setIsPlanOpen] = useState(false);

  const openPaywall = useCallback(() => setIsPlanOpen(true), []);

  const planGate = (
    <DeployPlanGateDialog isOpen={isPlanOpen} onOpenChange={setIsPlanOpen} from="deploy" />
  );

  return { gated, openPaywall, planGate };
}
