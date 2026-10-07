"use client";

import type { DeployCheckoutOrigin } from "@/lib/navigation/routes";
import { PlansScreen } from "../../settings/billing/components/plans-screen";

type Props = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  from: DeployCheckoutOrigin;
};

export function DeployPlanGateDialog({ isOpen, onOpenChange, from }: Props) {
  return <PlansScreen open={isOpen} onOpenChange={onOpenChange} reason="deploy" from={from} />;
}
