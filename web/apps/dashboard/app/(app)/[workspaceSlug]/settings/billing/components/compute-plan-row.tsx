"use client";

import { formatDollars } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { Cube } from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  Skeleton,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { CancelComputeDialog, CancelPlanLink } from "./cancel-actions";
import {
  AllPlansInclude,
  ComputePlanConfirmDialog,
  ComputePlanDialog,
  ComputePlanRows,
  CreditsInfoStrip,
} from "./compute-plan-picker";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { creditLabel, reconcileDeployInvoice } from "./deploy-invoice";

type ComputePlanRowProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  /** Primary styling, used only while nothing at all is subscribed. */
  emphasize: boolean;
  /** Open the plan picker on mount (post-checkout intent hand-off). */
  autoOpenPlanModal?: boolean;
};

/**
 * Compute in the Plans card: the tier sits with the price, and the subtitle is
 * the usage credit the fee buys. What the next invoice reconciles to lives on the
 * upcoming-invoice card; the per-meter quantities behind it live on Usage.
 */
export function ComputePlanRow({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  emphasize,
  autoOpenPlanModal = false,
}: ComputePlanRowProps) {
  const trpcUtils = trpc.useUtils();
  const [isPlanModalOpen, setPlanModalOpen] = useState(autoOpenPlanModal);
  const [pendingPlan, setPendingPlan] = useState<DeployPlan | null>(null);
  const [isStartingCheckout, setIsStartingCheckout] = useState(false);
  const [isCancelOpen, setCancelOpen] = useState(false);

  const { data: subscription, isLoading: subscriptionLoading } =
    trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    { staleTime: 60_000, trpc: { context: { skipBatch: true } } },
  );

  const currentPlan = subscription?.plan ?? null;

  const { data: deployCredit } = trpc.stripe.getDeployCredit.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });
  const { data: usage } = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async (result) => {
      if (result.kind === "payment_required") {
        window.location.assign(result.paymentUrl);
        return;
      }
      setPendingPlan(null);
      setPlanModalOpen(false);
      toast.success("Compute plan changed");
      await Promise.all([
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
        trpcUtils.stripe.getDeployCredit.invalidate(),
        trpcUtils.billing.queryDeployUsage.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.stripe.getDeploySubscription.refetch(),
      ]);
    },
    onError: (err) => toast.error(err.message),
  });

  if (subscriptionLoading || plansLoading) {
    return (
      <Item>
        <ItemMedia className="bg-orangeA-3 text-orange-11">
          <Cube />
        </ItemMedia>
        <ItemContent className="gap-1">
          <Skeleton className="h-3.5 w-20" />
          <Skeleton className="h-3 w-32" />
        </ItemContent>
      </Item>
    );
  }

  // Deploy billing not configured server-side: hide the row entirely.
  if (plansData && !plansData.configured) {
    return null;
  }

  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);
  const planFee = currentPlanOption?.amount ?? null;
  const usageAmount = usage?.grossCents ?? null;

  const invoice = reconcileDeployInvoice({
    usageCents: usageAmount,
    projectedUsageCents: usage?.projectedGrossCents ?? null,
    planFeeCents: planFee,
    grantedCreditCents: deployCredit?.includedCreditCents ?? null,
    previewTotalCents: null,
  });

  const description = currentPlan
    ? planFee === null
      ? "The plan fee includes usage credits; usage beyond them is billed on top."
      : creditLabel(planFee, invoice)
    : "Choose a plan to start deploying on Unkey";

  const warningFor = (option: (typeof plans)[number]): string | null =>
    option.amount !== null && usageAmount !== null && usageAmount > option.amount
      ? `Your usage this period (${formatDollars(usageAmount)}) already exceeds the ${formatDollars(
          option.amount,
        )} of monthly credits ${option.name} includes. This period keeps your current credits; from next period, usage at this level is billed as overage.`
      : null;

  const pendingPlanOption = plans.find((p) => p.plan === pendingPlan);
  const commitPending = () => {
    if (!pendingPlan) {
      return;
    }
    if (currentPlan) {
      change.mutate({ plan: pendingPlan });
      return;
    }
    setIsStartingCheckout(true);
    window.location.assign(
      routes.settings.stripe.checkout({
        workspaceSlug,
        intent: "deploy",
        plan: pendingPlan,
        from: "billing",
      }),
    );
  };

  return (
    <>
      <Item>
        <ItemMedia className="bg-orangeA-3 text-orange-11">
          <Cube />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Compute</ItemTitle>
          <ItemDescription>{description}</ItemDescription>
        </ItemContent>
        <ItemActions className="gap-4">
          {currentPlan && planFee !== null ? (
            <span className="w-48 text-right tabular-nums">
              <span className="text-gray-11">{currentPlanOption?.name ?? currentPlan} - </span>
              <span className="font-medium text-gray-12">
                {formatDollars(planFee)}/{currentPlanOption?.interval ?? "month"}
              </span>
            </span>
          ) : null}
          <span className="flex w-32 justify-end">
            {currentPlan ? (
              <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                <span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!isAdmin}
                    onClick={() => setPlanModalOpen(true)}
                  >
                    Change
                  </Button>
                </span>
              </InfoTooltip>
            ) : (
              <InfoTooltip
                content={hasPaymentMethod ? ADMIN_ONLY_TOOLTIP : "Add a payment method first"}
                disabled={isAdmin && hasPaymentMethod}
                asChild
              >
                <span>
                  <Button
                    variant={emphasize ? "primary" : "outline"}
                    size="sm"
                    disabled={!isAdmin || !hasPaymentMethod}
                    onClick={() => setPlanModalOpen(true)}
                  >
                    Choose a plan
                  </Button>
                </span>
              </InfoTooltip>
            )}
          </span>
        </ItemActions>
      </Item>

      <ComputePlanDialog
        isOpen={isPlanModalOpen}
        onOpenChange={setPlanModalOpen}
        title={currentPlan ? "Change Compute plan" : "Choose a Compute plan"}
        subTitle="The monthly plan fee includes the same amount of usage credits; usage beyond them is billed on top."
      >
        <ComputePlanRows
          plans={plans}
          currentPlan={currentPlan}
          currentPlanAmount={currentPlan ? planFee : null}
          submittingPlan={
            isStartingCheckout
              ? pendingPlan
              : change.isLoading
                ? (change.variables?.plan ?? null)
                : null
          }
          onSelect={(plan) => {
            setPendingPlan(plan);
            setPlanModalOpen(false);
          }}
          warningFor={warningFor}
          disabledReason={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
        />
        <AllPlansInclude />
        <CreditsInfoStrip />
        {currentPlan ? (
          <div className="flex justify-center pt-1">
            <CancelPlanLink
              isAdmin={isAdmin}
              onClick={() => {
                setPlanModalOpen(false);
                setCancelOpen(true);
              }}
            >
              Cancel Compute plan
            </CancelPlanLink>
          </div>
        ) : null}
      </ComputePlanDialog>

      <CancelComputeDialog open={isCancelOpen} onOpenChange={setCancelOpen} />

      <ComputePlanConfirmDialog
        plan={pendingPlanOption ?? null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingPlan(null);
          }
        }}
        onConfirm={commitPending}
        isLoading={isStartingCheckout || change.isLoading}
        currentPlanName={currentPlan ? (currentPlanOption?.name ?? currentPlan) : undefined}
        note="Takes effect immediately. Upgrades are charged now and add the difference as usage credits; downgrades keep this period's credits, with the new fee starting next period."
      />
    </>
  );
}
