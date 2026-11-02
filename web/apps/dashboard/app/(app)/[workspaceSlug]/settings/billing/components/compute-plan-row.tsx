"use client";

import { usePaymentIssues } from "@/components/billing/payment-issues";
import { trpc } from "@/lib/trpc/client";
import { useState } from "react";
import { periodCredit } from "./deploy-invoice";
import {
  PlanName,
  PlanPrice,
  PlanRowAction,
  PlanRowLayout,
  PlanRowMessage,
  PlanRowSkeleton,
} from "./plan-row";
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
    return <PlanRowSkeleton product="compute" />;
  }

  if (subscriptionError || plansError) {
    return (
      <PlanRowMessage product="compute">
        Compute plans could not be loaded. Reload the page or contact support@unkey.com.
      </PlanRowMessage>
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
      <PlanRowLayout
        product="compute"
        paymentFailed={paymentFailed}
        name={<PlanName>{currentPlan ? (currentPlanOption?.name ?? currentPlan) : null}</PlanName>}
        price={
          <PlanPrice
            feeCents={currentPlan ? planFee : null}
            interval={currentPlanOption?.interval ?? "month"}
            usageCreditCents={credit?.cents ?? null}
            usageCreditProrated={credit?.prorated ?? false}
          />
        }
        action={
          <PlanRowAction
            product="compute"
            paymentFailed={paymentFailed}
            isAdmin={isAdmin}
            hasPlan={currentPlan !== null}
            emphasize={emphasize}
            onClick={() => setPlanModalOpen(true)}
            chooseLabel="Choose a plan"
          />
        }
      />

      <PlansScreen open={isPlanModalOpen} onOpenChange={setPlanModalOpen} reason="compute-plan" />
    </>
  );
}
