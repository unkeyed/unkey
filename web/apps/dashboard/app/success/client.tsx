"use client";
import { useInvalidateBillingQueries } from "@/hooks/use-payment-issues";
import { type CheckoutOutcome, checkoutReturnPath } from "@/lib/billing/upgrade-result";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { PlansScreen } from "../(app)/[workspaceSlug]/settings/billing/components/plans-screen";

type Props = {
  workSpaceSlug?: string;
  outcome: CheckoutOutcome;
  showPlanSelection?: boolean;
  intent?: string;
  plan?: string;
  from?: string;
  returnTo?: string;
};

export function SuccessClient({
  workSpaceSlug,
  outcome,
  showPlanSelection,
  intent,
  plan,
  from,
  returnTo,
}: Props) {
  const router = useRouter();
  const invalidateBilling = useInvalidateBillingQueries();
  const choosingPlan = Boolean(showPlanSelection && workSpaceSlug);
  const [showModal, setShowModal] = useState(choosingPlan);

  useEffect(() => {
    if (choosingPlan) {
      return;
    }
    if (!workSpaceSlug) {
      router.replace("/");
      return;
    }

    router.replace(
      checkoutReturnPath({ workspaceSlug: workSpaceSlug, outcome, intent, plan, from, returnTo }),
    );
  }, [router, workSpaceSlug, choosingPlan, outcome, intent, plan, from, returnTo]);

  if (!choosingPlan || !workSpaceSlug) {
    return null;
  }

  const leave = async () => {
    setShowModal(false);
    await invalidateBilling();
    router.push(routes.settings.billing({ workspaceSlug: workSpaceSlug }));
  };

  return (
    <PlansScreen
      open={showModal}
      onOpenChange={(open) => {
        if (open) {
          setShowModal(true);
          return;
        }
        void leave();
      }}
      reason="choose-plan"
    />
  );
}
