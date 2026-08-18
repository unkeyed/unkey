"use client";

import { formatDollars, formatPrice } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { Cube } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useState } from "react";
import { AdminGate } from "./admin-gate";
import { CancelComputeDialog, CancelPlanLink } from "./cancel-actions";
import {
  AllPlansInclude,
  ComputePlanConfirmDialog,
  ComputePlanDialog,
  ComputePlanRows,
  CreditsInfoStrip,
} from "./compute-plan-picker-v2";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { periodCredit } from "./deploy-invoice";
import { PlanTableRow, PlanTableRowMessage, PlanTableRowSkeleton } from "./plan-table-row";

const NEEDS_PAYMENT_TOOLTIP = "Add a payment method first";
const MEDIA = "bg-orangeA-3 text-orange-11";

type ComputePlanRowProps = {
  isAdmin: boolean | undefined;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  emphasize: boolean;
  autoOpenPlanModal?: boolean;
};

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
      ]);
    },
    onError: (err) => toast.error(err.message),
  });

  if (subscriptionLoading || plansLoading) {
    return <PlanTableRowSkeleton icon={<Cube />} mediaClassName={MEDIA} />;
  }

  if (subscriptionError || plansError) {
    return (
      <PlanTableRowMessage>
        Compute plans could not be loaded. Reload the page or contact support@unkey.com.
      </PlanTableRowMessage>
    );
  }

  if (plansData && !plansData.configured) {
    return null;
  }

  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);
  const planFee = currentPlanOption?.amount ?? null;
  const usageAmount = usage?.grossCents ?? null;

  const credit = periodCredit(planFee, deployCredit?.includedCreditCents ?? null);

  const warningFor = (option: (typeof plans)[number]): string | null =>
    option.amount !== null && usageAmount !== null && usageAmount > option.amount
      ? `Your usage this period (${formatPrice(usageAmount)}) already exceeds the ${formatDollars(
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
      <PlanTableRow
        icon={<Cube />}
        mediaClassName={MEDIA}
        title="Compute"
        planName={currentPlan ? (currentPlanOption?.name ?? currentPlan) : null}
        feeCents={currentPlan ? planFee : null}
        interval={currentPlanOption?.interval ?? "month"}
        usageCreditCents={credit?.cents ?? null}
        action={
          currentPlan ? (
            <AdminGate isAdmin={isAdmin}>
              {(disabled) => (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={disabled}
                  onClick={() => setPlanModalOpen(true)}
                >
                  Change
                </Button>
              )}
            </AdminGate>
          ) : (
            <AdminGate
              isAdmin={isAdmin}
              blocked={!hasPaymentMethod}
              blockedReason={NEEDS_PAYMENT_TOOLTIP}
            >
              {(disabled) => (
                <Button
                  variant={emphasize ? "primary" : "outline"}
                  size="sm"
                  disabled={disabled}
                  onClick={() => setPlanModalOpen(true)}
                >
                  Choose a plan
                </Button>
              )}
            </AdminGate>
          )
        }
      />

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
