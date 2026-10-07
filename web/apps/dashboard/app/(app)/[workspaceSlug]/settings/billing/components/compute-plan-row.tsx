"use client";

import { trpc } from "@/lib/trpc/client";
import { IconCubeOutline18 } from "@unkey/icons";
import { Item, ItemActions, ItemContent, ItemMedia, ItemTitle, Skeleton } from "@unkey/ui";
import { useState } from "react";
import { periodCredit } from "./deploy-invoice";
import { PlanName, PlanPrice, PlanRowAction } from "./plan-row";
import { PlansScreen } from "./plans-screen";

function ProductCell() {
  return (
    <>
      <ItemMedia className="bg-orangeA-3 text-orange-11">
        <IconCubeOutline18 />
      </ItemMedia>
      <ItemContent>
        <ItemTitle className="truncate">Compute</ItemTitle>
      </ItemContent>
    </>
  );
}

type ComputePlanRowProps = {
  isAdmin: boolean | undefined;
  emphasize: boolean;
};

export function ComputePlanRow({ isAdmin, emphasize }: ComputePlanRowProps) {
  const [isPlanModalOpen, setPlanModalOpen] = useState(false);

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
        <ProductCell />
        <ItemActions className="gap-3">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-3 w-36" />
          <span className="w-20" />
        </ItemActions>
      </Item>
    );
  }

  if (subscriptionError || plansError) {
    return (
      <Item>
        <ProductCell />
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
        <ProductCell />
        <ItemActions className="gap-3">
          <PlanName>{currentPlan ? (currentPlanOption?.name ?? currentPlan) : null}</PlanName>
          <PlanPrice
            feeCents={currentPlan ? planFee : null}
            interval={currentPlanOption?.interval ?? "month"}
            usageCreditCents={credit?.cents ?? null}
            usageCreditProrated={credit?.prorated ?? false}
          />
          <span className="flex w-20 justify-end">
            <PlanRowAction
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
