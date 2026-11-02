"use client";

import { usePaymentIssues } from "@/components/billing/payment-issues";
import { currentApiProduct } from "@/lib/billing/api-plan";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { useState } from "react";
import { PlanName, PlanPrice, PlanRowAction, PlanRowLayout, PlanRowSkeleton } from "./plan-row";
import { PlansScreen } from "./plans-screen";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type ApiPlanRowProps = {
  isAdmin: boolean | undefined;
  emphasize: boolean;
  /** Undefined while the billing info loads */
  billing: BillingInfo | undefined;
};

export function ApiPlanRow({ isAdmin, emphasize, billing }: ApiPlanRowProps) {
  const [showPlanModal, setShowPlanModal] = useState(false);
  const paymentFailed = usePaymentIssues().includes("api");

  if (!billing) {
    return <PlanRowSkeleton product="api" />;
  }

  const currentProduct = currentApiProduct(billing);

  return (
    <>
      <PlanRowLayout
        product="api"
        paymentFailed={paymentFailed}
        name={<PlanName>{currentProduct ? currentProduct.name : "Free"}</PlanName>}
        price={<PlanPrice feeCents={(currentProduct?.dollar ?? 0) * 100} />}
        action={
          <PlanRowAction
            product="api"
            paymentFailed={paymentFailed}
            isAdmin={isAdmin}
            hasPlan={currentProduct !== undefined}
            emphasize={emphasize}
            onClick={() => setShowPlanModal(true)}
            chooseLabel="Upgrade"
          />
        }
      />

      <PlansScreen open={showPlanModal} onOpenChange={setShowPlanModal} reason="api-plan" />
    </>
  );
}
