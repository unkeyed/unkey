"use client";

import { usePaymentIssues } from "@/components/billing/payment-issues";
import { trpc } from "@/lib/trpc/client";
import { Item, ItemActions } from "@unkey/ui";
import { useState } from "react";
import { periodCredit } from "./deploy-invoice";
import { PlanCellsSkeleton, PlanName, PlanPrice, PlanProductCell, PlanRowAction } from "./plan-row";
import { PlansScreen } from "./plans-screen";

type ComputePlanRowProps = {
  isAdmin: boolean | undefined;
  emphasize: boolean;
};

export function ComputePlanRow({ isAdmin, emphasize }: ComputePlanRowProps) {
  const [isPlanModalOpen, setPlanModalOpen] = useState(false);
  const paymentFailed = usePaymentIssues().includes("compute");

  const {
    data: subscription,
    isLoading: subscriptionLoading,
    isError: subscriptionError,
  } = trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const {
    data: plansData,
    isLoading: plansLoading,
    isError: plansError,
  } = trpc.stripe.getDeployPlans.useQuery(undefined, {
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });

  const currentPlan = subscription?.plan ?? null;

  const { data: deployCredit } = trpc.stripe.getDeployCredit.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  if (subscriptionLoading || plansLoading) {
    return (
      <Item>
        <PlanProductCell product="compute" />
        <ItemActions className="gap-3">
          <PlanCellsSkeleton />
        </ItemActions>
      </Item>
    );
  }

  if (subscriptionError || plansError) {
    return (
      <Item>
        <PlanProductCell product="compute" />
        <p className="text-sm text-gray-11">
          Compute plans could not be loaded. Reload the page or contact support@unkey.com.
        </p>
      </Item>
    );
  }

  if (plansData && !plansData.configured) {
    return null;
  }

  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);
  const planFee = currentPlanOption?.amount ?? null;

  const credit = periodCredit(planFee, deployCredit?.includedCreditCents ?? null);

  return (
    <>
      <Item>
        <PlanProductCell product="compute" paymentFailed={paymentFailed} />
        <ItemActions className="gap-3">
          <PlanName>{currentPlan ? (currentPlanOption?.name ?? currentPlan) : null}</PlanName>
          <PlanPrice
            feeCents={currentPlan ? planFee : null}
            interval={currentPlanOption?.interval ?? "month"}
            usageCreditCents={credit?.cents ?? null}
            usageCreditProrated={credit?.prorated ?? false}
          />
          <span className="flex w-(--plan-action-w) justify-end">
            <PlanRowAction
              product="compute"
              paymentFailed={paymentFailed}
              isAdmin={isAdmin}
              hasPlan={currentPlan !== null}
              emphasize={emphasize}
              onClick={() => setPlanModalOpen(true)}
              chooseLabel="Choose a plan"
            />
          </span>
        </ItemActions>
      </Item>

      <PlansScreen open={isPlanModalOpen} onOpenChange={setPlanModalOpen} reason="compute-plan" />
    </>
  );
}
