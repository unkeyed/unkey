"use client";

import { DEPLOY_METER_RATE_LABELS, priceDeployMetersCents } from "@/lib/billing/deployPricing";
import { formatCompactQuantity, formatDollars, formatPrice } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { IconCubeOutline18 } from "nucleo-ui-outline-18";
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

/** Matches the billing summary strip's period formatting, e.g. "Aug 1". */
function formatRenewalDate(millis: number): string {
  return new Date(millis).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

type DeployProductCardProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
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
  workspaceSlug,
  autoOpenPlanModal = false,
}) => {
  const trpcUtils = trpc.useUtils();
  const [isPlanModalOpen, setPlanModalOpen] = useState(autoOpenPlanModal);
  const [isCancelOpen, setCancelOpen] = useState(false);
  const [pendingPlan, setPendingPlan] = useState<DeployPlan | null>(null);
  const [isStartingCheckout, setIsStartingCheckout] = useState(false);

  const { data: subscription, isLoading: subscriptionLoading } =
    trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    {
      staleTime: 60_000,
      trpc: { context: { skipBatch: true } },
    },
  );

  const currentPlan = subscription?.plan ?? null;

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, { staleTime: 30_000 });
  const suspended = budget?.suspended ?? false;

  const { data: deployCredit } = trpc.stripe.getDeployCredit.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  // For the renewal date. The billing summary strip already fetches this, so on
  // this page it's a cache hit.
  const { data: upcomingInvoice } = trpc.stripe.getUpcomingInvoice.useQuery(undefined, {
    enabled: Boolean(currentPlan) && hasPaymentMethod,
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
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

  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async (result) => {
      if (result.kind === "payment_required") {
        window.location.assign(result.paymentUrl);
        return;
      }
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

  // The plan fee actually invoiced for THIS period, which after a mid-cycle
  // change is the prorated amount, not the catalog price. grantDeployCreditsForInvoice
  // grants credit equal to netDeployFee, so the period's grant is the fee it was
  // charged for: reading the fee off the grant keeps this period's total honest
  // without a second Stripe round-trip. Falls back to the catalog fee when there
  // is no grant to read (query still in flight, or a grant the webhook never
  // wrote) so a missing grant understates the credit rather than the fee.
  const periodFeeCents =
    includedCreditCents !== null && includedCreditCents > 0 ? includedCreditCents : planFee;

  // A mid-cycle plan change prorates both the fee and the credit, so this
  // period's numbers are not the catalog ones. Say so rather than quoting a full
  // period the customer has not been billed for yet.
  const feeProrated = periodFeeCents !== null && planFee !== null && periodFeeCents !== planFee;

  // Usage beyond the included credit: the only usage-driven charge on the bill,
  // and the only part of this period still to come. The credit is not a discount
  // applied to usage, it is the allowance the period's fee already paid for.
  const overageCents =
    usageAmount !== null && includedCreditCents !== null
      ? Math.max(0, usageAmount - includedCreditCents)
      : null;
  // Headroom left in the allowance. Shown instead of a "credit applied" line:
  // the credit is not a discount that grows with usage, so the useful number is
  // what is left of it, and it keeps the plan picker's credit vocabulary on the
  // card without implying the credit is subtracted from the total.
  const creditRemainingCents =
    usageAmount !== null && includedCreditCents !== null
      ? Math.max(0, includedCreditCents - usageAmount)
      : null;

  const projectedAmount = usage?.projectedGrossCents ?? null;
  const projectedOverageCents =
    projectedAmount !== null && includedCreditCents !== null
      ? Math.max(0, projectedAmount - includedCreditCents)
      : null;

  // What this period costs in total: the fee it was invoiced plus whatever usage
  // spills past the included credit. "Projected" prices usage extrapolated to
  // period end, using the same local pricing as the spend bar and the spend cap,
  // so the numbers on the card never disagree.
  const currentBillCents =
    periodFeeCents !== null && overageCents !== null ? periodFeeCents + overageCents : null;

  // The metered usage for a period bills on the invoice that finalizes at the
  // start of the next one (see EXPIRY_GRACE_SECONDS), so period end is both the
  // renewal date and when this period's overage lands.
  const renewsAtMillis = upcomingInvoice?.deploy?.periodEnd ?? null;

  // What the next invoice charges: this period's overage plus the next period's
  // full fee, the two things that land on the same document. Stripe's own preview
  // is authoritative (it applies the credit and any proration we don't model), so
  // prefer it and fall back to the local sum when it hasn't loaded or the
  // workspace has no payment method yet.
  const nextInvoiceCents =
    upcomingInvoice?.deploy?.total ??
    (overageCents !== null && planFee !== null ? overageCents + planFee : null);

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

  const submittingPlan = isStartingCheckout
    ? pendingPlan
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
    } else {
      setIsStartingCheckout(true);
      window.location.assign(
        routes.settings.stripe.checkout({
          workspaceSlug,
          intent: "deploy",
          plan: pendingPlan,
          from: "billing",
        }),
      );
    }
  };

  return (
    <>
      <ProductCard
        icon={<IconCubeOutline18 className="size-4" />}
        iconClassName="bg-orangeA-3 text-orange-11"
        className="[&>div:nth-child(2)]:border-t-0 [&>div:nth-child(2)]:pt-0"
        name="Compute"
        tag={currentPlan ? (currentPlanOption?.name ?? currentPlan) : undefined}
        badge={currentPlan && suspended ? <ComputePausedBadge /> : undefined}
        subtitle={
          currentPlan
            ? planFee !== null && includedCreditCents !== null
              ? feeProrated
                ? `${formatDollars(planFee)}/${currentPlanOption?.interval ?? "month"}, prorated to ${formatDollars(includedCreditCents)} of fee and credits for this period`
                : `${formatDollars(planFee)}/${currentPlanOption?.interval ?? "month"}, includes ${formatDollars(planFee)} of usage credits`
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
                    <p className="font-normal text-[13px] text-gray-12 tabular-nums">
                      {stat.value}
                    </p>
                    <p className="text-[12px] text-gray-10 tabular-nums">
                      {formatPrice(stat.cost)}
                    </p>
                  </div>
                ))}
              </div>
            ) : null}
            {currentBillCents !== null &&
            planFee !== null &&
            periodFeeCents !== null &&
            includedCreditCents !== null &&
            overageCents !== null ? (
              <div className="flex flex-col gap-1.5">
                {/* Reconcile the next invoice, because that is the number people
                    open this page to find, and it is the one Stripe shows: this
                    period's overage plus the next period's full fee, the two
                    charges that land on the same document. Only those two rows are
                    money owed, so only they carry normal weight. This period's fee
                    and credit balance are context, dimmed, and marked paid: they
                    explain where the overage came from without reading as charges,
                    which is what made a prorated period look like it undercharged.
                    The credit reads as what is LEFT rather than what was applied,
                    because a credit subtracted from a total is the reading that
                    makes a $0 line look like a missing grant. */}
                <div className="flex items-baseline justify-between gap-4">
                  <span className="text-[13px] text-gray-10">
                    Plan fee
                    {feeProrated ? (
                      <span className="ml-1.5 text-[12px] text-gray-9">prorated</span>
                    ) : null}
                  </span>
                  <span className="text-[13px] text-gray-9 tabular-nums">
                    {formatDollars(periodFeeCents)} paid
                  </span>
                </div>
                <div className="flex items-baseline justify-between gap-4">
                  <span className="text-[13px] text-gray-10">Usage</span>
                  <span className="text-[13px] text-gray-9 tabular-nums">
                    {formatPrice(usageAmount ?? 0)}
                  </span>
                </div>
                {includedCreditCents > 0 && creditRemainingCents !== null ? (
                  <div className="flex items-baseline justify-between gap-4">
                    <span className="text-[13px] text-gray-10">Included credit</span>
                    <span className="text-[13px] text-gray-9 tabular-nums">
                      {formatPrice(creditRemainingCents)} of {formatDollars(includedCreditCents)}{" "}
                      remaining
                    </span>
                  </div>
                ) : null}
                {includedCreditCents > 0 ? (
                  <div className="mt-1 flex items-baseline justify-between gap-4 border-grayA-3 border-t pt-2">
                    <span className="text-[13px] text-gray-10">
                      Overage
                      <span className="ml-1.5 text-[12px] text-gray-9">usage past credit</span>
                    </span>
                    <span className="text-[13px] text-gray-11 tabular-nums">
                      {formatPrice(overageCents)}
                    </span>
                  </div>
                ) : null}
                <div className="flex items-baseline justify-between gap-4">
                  <span className="text-[13px] text-gray-10">
                    Next plan fee
                    <span className="ml-1.5 text-[12px] text-gray-9">
                      {currentPlanOption?.interval ?? "month"} ahead
                    </span>
                  </span>
                  <span className="text-[13px] text-gray-11 tabular-nums">
                    {formatDollars(planFee)}
                  </span>
                </div>
                <div className="mt-1 flex items-baseline justify-between gap-4 border-grayA-3 border-t pt-2">
                  <span className="text-[13px] text-gray-12">
                    <InfoTooltip
                      asChild
                      position={{ side: "top", align: "start" }}
                      content={
                        <div className="flex max-w-[240px] flex-col gap-2 text-[12px]">
                          <div className="flex flex-col gap-0.5">
                            <p className="font-normal text-gray-12">How this is calculated</p>
                            <p className="text-gray-11">
                              This period's {formatDollars(periodFeeCents)} fee is already invoiced
                              and covers {formatDollars(includedCreditCents)} of usage. The next
                              invoice charges what your usage went past that, plus the coming
                              period's fee.
                            </p>
                            <p className="text-gray-11 tabular-nums">
                              {formatPrice(overageCents)} + {formatDollars(planFee)} ={" "}
                              {formatPrice(nextInvoiceCents ?? overageCents + planFee)}
                            </p>
                          </div>
                          <div className="flex flex-col gap-1 border-grayA-4 border-t pt-2">
                            <p className="font-normal text-gray-12">Usage rates</p>
                            <ul className="flex flex-col gap-0.5">
                              {DEPLOY_METER_RATE_LABELS.map((r) => (
                                <li
                                  key={r.label}
                                  className="flex justify-between gap-3 text-gray-11"
                                >
                                  <span>{r.label}</span>
                                  <span className="tabular-nums">{r.rate}</span>
                                </li>
                              ))}
                            </ul>
                          </div>
                        </div>
                      }
                    >
                      <span className="cursor-help underline decoration-dotted decoration-grayA-6 underline-offset-2">
                        Next invoice
                        {renewsAtMillis !== null ? ` · ${formatRenewalDate(renewsAtMillis)}` : ""}
                      </span>
                    </InfoTooltip>
                    {/* Projected adds the usage still expected before the period
                        closes, since the overage row only counts what has accrued. */}
                    {projectedOverageCents !== null &&
                    overageCents !== null &&
                    nextInvoiceCents !== null &&
                    projectedOverageCents > overageCents ? (
                      <span className="ml-1.5 text-[12px] text-gray-9">
                        (~
                        {formatPrice(nextInvoiceCents + (projectedOverageCents - overageCents))}{" "}
                        projected)
                      </span>
                    ) : null}
                  </span>
                  <span className="font-normal text-[15px] text-gray-12 tabular-nums">
                    {nextInvoiceCents !== null ? formatPrice(nextInvoiceCents) : "—"}
                  </span>
                </div>
                <p className="text-[12px] text-gray-9">
                  This period's {formatDollars(periodFeeCents)} fee is already paid. Total cost for
                  this period is {formatPrice(currentBillCents)}.
                </p>
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
        isLoading={isStartingCheckout || change.isLoading}
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
