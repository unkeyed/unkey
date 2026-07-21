"use client";

import { priceDeployMetersCents } from "@/lib/billing/deployPricing";
import { formatCompactQuantity, formatDollars } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { Cube } from "@unkey/icons";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { useState } from "react";
import { ComputePausedBadge } from "./compute-paused";
import {
  AllPlansInclude,
  ComputePlanConfirmDialog,
  ComputePlanDialog,
  ComputePlanRows,
  CreditsInfoStrip,
} from "./compute-plan-picker";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { ProductCard } from "./product-card";
import { SpendManagement } from "./spend-management";

type DeployProductCardProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  /** Open the plan picker on mount (post-checkout intent hand-off). */
  autoOpenPlanModal?: boolean;
};

/**
 * The Deploy product card, the page's hero: current plan and fee, the spend
 * budget, and the per-meter month-to-date quantities the spend is made of.
 * Without a plan it's the subscribe entry point. Cancelling is a quiet footer
 * link with a confirmation dialog, not a danger zone.
 */
export const DeployProductCard: React.FC<DeployProductCardProps> = ({
  isAdmin,
  hasPaymentMethod,
  autoOpenPlanModal = false,
}) => {
  const trpcUtils = trpc.useUtils();
  const [isPlanModalOpen, setPlanModalOpen] = useState(autoOpenPlanModal);
  const [isCancelOpen, setCancelOpen] = useState(false);
  const [pendingPlan, setPendingPlan] = useState<DeployPlan | null>(null);

  const { data: subscription, isLoading: subscriptionLoading } =
    trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    { staleTime: 60_000 },
  );

  const currentPlan = subscription?.plan ?? null;

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, { staleTime: 30_000 });
  const suspended = budget?.suspended ?? false;

  const { data: deployCredit } = trpc.stripe.getDeployCredit.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
  });

  // Usage is only worth fetching (and rendering) once there is a plan.
  const { data: usage } = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

  const revalidate = async () => {
    await Promise.all([
      trpcUtils.stripe.getDeploySubscription.invalidate(),
      trpcUtils.stripe.getDeployEntitlement.invalidate(),
      trpcUtils.stripe.getUpcomingInvoice.invalidate(),
      trpcUtils.billing.queryDeployUsage.invalidate(),
      trpcUtils.workspace.getCurrent.invalidate(),
      trpcUtils.stripe.getDeploySubscription.refetch(),
    ]);
  };

  const subscribe = trpc.stripe.subscribeDeploy.useMutation({
    onSuccess: async () => {
      setPendingPlan(null);
      setPlanModalOpen(false);
      toast.success("Subscribed to Compute");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async () => {
      setPendingPlan(null);
      setPlanModalOpen(false);
      toast.success("Compute plan changed");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const cancel = trpc.stripe.cancelDeploy.useMutation({
    onSuccess: async () => {
      setCancelOpen(false);
      toast.info("Compute cancelled");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  if (subscriptionLoading || plansLoading) {
    return <div className="h-[150px] w-full animate-pulse rounded-lg bg-grayA-3" />;
  }

  // Deploy billing not configured server-side: hide the card entirely.
  if (plansData && !plansData.configured) {
    return null;
  }

  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);

  // Gross month-to-date usage in cents, priced the same way the spend-cap
  // worker prices it, so the budget bar tracks what the cap enforces.
  const usageAmount = usage?.grossCents ?? null;

  // The plan's recurring fee, from the plan catalog.
  const planFee = currentPlanOption?.amount ?? null;

  // Included usage credit for the current period, from Stripe (cached). A
  // mid-cycle plan change prorates the grant, so the credit is not always equal
  // to the catalog fee; reading the granted amount is what keeps the estimate
  // matching the invoice. Falls back to the plan fee before the query resolves,
  // which is the steady-state (full clean month) value.
  const includedCreditCents = deployCredit?.includedCreditCents ?? planFee;

  // Estimated bill: the plan fee plus usage beyond the included credits. "This
  // month" prices month-to-date usage; "projected" prices usage extrapolated to
  // month end. Both use the same local pricing as the spend bar and the spend
  // cap, so the numbers on the card never disagree.
  const projectedAmount = usage?.projectedGrossCents ?? null;
  const currentBillCents =
    planFee !== null && includedCreditCents !== null && usageAmount !== null
      ? planFee + Math.max(0, usageAmount - includedCreditCents)
      : null;
  const projectedBillCents =
    planFee !== null && includedCreditCents !== null && projectedAmount !== null
      ? planFee + Math.max(0, projectedAmount - includedCreditCents)
      : null;

  // Price each meter locally (same rates as the spend bar) so every usage line
  // shows what it contributes to the bill; the parts sum to the gross above.
  const meterCosts = usage ? priceDeployMetersCents(usage) : null;
  const meterStats =
    usage && meterCosts
      ? [
          {
            label: "CPU",
            value: `${formatCompactQuantity(usage.cpuSeconds / 3600)} hrs`,
            cost: meterCosts.cpu,
            hint: "vCPU time your workloads ran, totalled across the period.",
          },
          {
            label: "Memory",
            value: `${formatCompactQuantity(usage.memoryGiBHours)} GiB-hrs`,
            cost: meterCosts.memory,
            hint: "Memory allocated over time, in GiB-hours. 1 GiB held for 1 hour is 1 GiB-hour.",
          },
          {
            label: "Egress",
            value: `${formatCompactQuantity(usage.egressGiB)} GiB`,
            cost: meterCosts.egress,
            hint: "Data your workloads sent out over the public network this period.",
          },
          {
            label: "Disk",
            value: `${formatCompactQuantity(usage.diskGiBHours)} GiB-hrs`,
            cost: meterCosts.disk,
            hint: "Disk reserved over time, in GiB-hours. 1 GiB reserved for 1 hour is 1 GiB-hour. Charged on size reserved, not reads, writes, or space used.",
          },
          {
            label: "Active keys",
            value: formatCompactQuantity(usage.activeKeys),
            cost: meterCosts.activeKeys,
            hint: "Distinct keys verified through the Deploy gateway this period.",
          },
        ]
      : null;

  const submittingPlan = subscribe.isLoading
    ? (subscribe.variables?.plan ?? null)
    : change.isLoading
      ? (change.variables?.plan ?? null)
      : null;

  const selectLabel = (option: (typeof plans)[number]): string => {
    if (!currentPlan || planFee === null || option.amount === null) {
      return "Select";
    }
    return option.amount > planFee ? "Upgrade" : "Downgrade";
  };

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
    } else {
      subscribe.mutate({ plan: pendingPlan });
    }
  };

  return (
    <>
      <ProductCard
        icon={<Cube iconSize="md-regular" />}
        iconClassName="bg-orangeA-3 text-orange-11"
        className="[&>div:nth-child(2)]:border-t-0 [&>div:nth-child(2)]:pt-0"
        name="Compute"
        tag={currentPlan ? (currentPlanOption?.name ?? currentPlan) : undefined}
        badge={currentPlan && suspended ? <ComputePausedBadge /> : undefined}
        subtitle={
          currentPlan
            ? planFee !== null
              ? `${formatDollars(planFee)}/${currentPlanOption?.interval ?? "month"}, includes ${formatDollars(planFee)} of usage credits`
              : "The plan fee includes usage credits; usage beyond them is billed on top."
            : "Run and scale your projects. Every plan includes usage credits equal to its fee."
        }
        action={
          currentPlan ? (
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <Button
                  variant="outline"
                  size="md"
                  disabled={!isAdmin}
                  onClick={() => setPlanModalOpen(true)}
                >
                  Change plan
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
                  variant="primary"
                  size="md"
                  disabled={!isAdmin || !hasPaymentMethod}
                  onClick={() => setPlanModalOpen(true)}
                >
                  Choose a plan
                </Button>
              </span>
            </InfoTooltip>
          )
        }
        footer={
          currentPlan ? (
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <button
                  type="button"
                  className="text-[13px] text-gray-9 transition-colors hover:text-gray-11 disabled:cursor-not-allowed"
                  disabled={!isAdmin}
                  onClick={() => setCancelOpen(true)}
                >
                  Cancel plan
                </button>
              </span>
            </InfoTooltip>
          ) : undefined
        }
      >
        {currentPlan ? (
          <div className="flex flex-col gap-4">
            {meterStats ? (
              <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg bg-grayA-3 sm:grid-cols-5">
                {meterStats.map((stat) => (
                  <div key={stat.label} className="bg-white px-3 py-2 first:pl-0 dark:bg-black">
                    <InfoTooltip content={stat.hint} asChild>
                      <p className="w-fit cursor-help text-[11px] text-gray-10 uppercase tracking-wide underline decoration-dotted decoration-grayA-6 underline-offset-2">
                        {stat.label}
                      </p>
                    </InfoTooltip>
                    <p className="font-medium text-[13px] text-gray-12 tabular-nums">
                      {stat.value}
                    </p>
                    <p className="text-[12px] text-gray-10 tabular-nums">
                      {formatDollars(stat.cost)}
                    </p>
                  </div>
                ))}
              </div>
            ) : null}
            {currentBillCents !== null && planFee !== null && includedCreditCents !== null ? (
              <div className="flex flex-col gap-1.5">
                {/* Reconcile the bill: plan fee plus gross usage less the
                    included credit equals the total. Usage is shown gross (the
                    same figure the spend limit plots) so the card never shows
                    two different usage numbers; the credit is its own line
                    instead of being pre-subtracted into usage. */}
                <div className="flex items-baseline justify-between gap-4">
                  <span className="text-[13px] text-gray-10">Plan fee</span>
                  <span className="text-[13px] text-gray-11 tabular-nums">
                    {formatDollars(planFee)}
                  </span>
                </div>
                <div className="flex items-baseline justify-between gap-4">
                  <span className="text-[13px] text-gray-10">Usage</span>
                  <span className="text-[13px] text-gray-11 tabular-nums">
                    {formatDollars(usageAmount ?? 0)}
                  </span>
                </div>
                {includedCreditCents > 0 ? (
                  <div className="flex items-baseline justify-between gap-4">
                    <span className="text-[13px] text-gray-10">Included credit</span>
                    <span className="text-[13px] text-gray-11 tabular-nums">
                      -{formatDollars(Math.min(usageAmount ?? 0, includedCreditCents))}
                    </span>
                  </div>
                ) : null}
                <div className="mt-1 flex items-baseline justify-between gap-4 border-grayA-3 border-t pt-2">
                  <span className="text-[13px] text-gray-12">Estimated bill this month</span>
                  <span className="font-medium text-[15px] text-gray-12 tabular-nums">
                    {formatDollars(currentBillCents)}
                  </span>
                </div>
                {projectedBillCents !== null && projectedBillCents > currentBillCents ? (
                  <div className="flex items-baseline justify-between gap-4">
                    <span className="text-[12px] text-gray-10">Projected by month end</span>
                    <span className="text-[12px] text-gray-11 tabular-nums">
                      ~{formatDollars(projectedBillCents)}
                    </span>
                  </div>
                ) : null}
              </div>
            ) : null}
            <SpendManagement usageCents={usageAmount} isAdmin={isAdmin} />
          </div>
        ) : null}
      </ProductCard>

      <ComputePlanDialog
        isOpen={isPlanModalOpen}
        onOpenChange={setPlanModalOpen}
        title={currentPlan ? "Change Compute plan" : "Choose a Compute plan"}
        subTitle="The monthly plan fee includes the same amount of usage credits; usage beyond them is billed on top."
      >
        <ComputePlanRows
          plans={plans}
          currentPlan={currentPlan}
          submittingPlan={submittingPlan}
          onSelect={(plan) => {
            setPendingPlan(plan);
            setPlanModalOpen(false);
          }}
          selectLabel={selectLabel}
          warningFor={warningFor}
          disabledReason={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
        />
        <AllPlansInclude />
        <CreditsInfoStrip />
      </ComputePlanDialog>

      <ComputePlanConfirmDialog
        plan={pendingPlanOption ?? null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingPlan(null);
          }
        }}
        onConfirm={commitPending}
        isLoading={subscribe.isLoading || change.isLoading}
        currentPlanName={currentPlan ? (currentPlanOption?.name ?? currentPlan) : undefined}
        note="Takes effect immediately. Upgrades are charged now and add the difference as usage credits; downgrades keep this period's credits, with the new fee starting next period."
      />

      <DialogContainer
        isOpen={isCancelOpen}
        onOpenChange={setCancelOpen}
        title="Cancel Compute"
        subTitle="Turn off Compute for this workspace"
        footer={
          <Button
            type="button"
            variant="primary"
            color="danger"
            size="xlg"
            className="w-full rounded-lg"
            loading={cancel.isLoading}
            onClick={() => cancel.mutate()}
          >
            Cancel Compute
          </Button>
        }
      >
        <div className="text-[13px] text-gray-11 leading-6">
          Cancelling stops Compute immediately: your deployments stop and no further usage is
          billed. Usage up to now is still charged, and the plan fee already paid is not refunded.
        </div>
      </DialogContainer>
    </>
  );
};
