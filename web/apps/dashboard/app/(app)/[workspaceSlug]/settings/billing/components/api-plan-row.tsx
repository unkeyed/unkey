"use client";

import { usePaymentIssues } from "@/components/billing/payment-issues";
import { currentApiProduct } from "@/lib/billing/api-plan";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { Item, ItemActions } from "@unkey/ui";
import { useState } from "react";
import { PlanCellsSkeleton, PlanName, PlanPrice, PlanProductCell, PlanRowAction } from "./plan-row";
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
    return (
      <Item>
        <PlanProductCell product="api" />
        <ItemActions className="gap-3">
          <PlanCellsSkeleton />
        </ItemActions>
      </Item>
    );
  }

  const currentProduct = currentApiProduct(billing);

  return (
    <>
      <Item>
        <PlanProductCell product="api" paymentFailed={paymentFailed} />
        <ItemActions className="gap-3">
          <PlanName>{currentProduct ? currentProduct.name : "Free"}</PlanName>
          <PlanPrice feeCents={(currentProduct?.dollar ?? 0) * 100} />
          <span className="flex w-(--plan-action-w) justify-end">
            <PlanRowAction
              product="api"
              paymentFailed={paymentFailed}
              isAdmin={isAdmin}
              hasPlan={currentProduct !== undefined}
              emphasize={emphasize}
              onClick={() => setShowPlanModal(true)}
              chooseLabel="Upgrade"
            />
          </span>
        </ItemActions>
      </Item>

      <PlansScreen open={showPlanModal} onOpenChange={setShowPlanModal} reason="api-plan" />
    </>
  );
}
