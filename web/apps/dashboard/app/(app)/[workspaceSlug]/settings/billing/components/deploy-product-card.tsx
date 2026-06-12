"use client";

import { formatDollars } from "@/lib/fmt";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { Cube } from "@unkey/icons";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { PlanChangeModal } from "./plan-change-modal";
import { ProductCard } from "./product-card";
import { UsageMeter } from "./usage-meter";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

function formatQuantity(value: number): string {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 1 }).format(value);
}

type DeployProductCardProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  /** Open the plan picker on mount (post-checkout intent hand-off). */
  autoOpenPlanModal?: boolean;
};

/**
 * The Deploy product card, the page's hero: current plan and fee, usage spend
 * against the included credits, and the per-meter month-to-date quantities the
 * spend is made of. Without a plan it's the subscribe entry point. Cancelling
 * is a quiet footer link with a confirmation dialog, not a danger zone.
 */
export const DeployProductCard: React.FC<DeployProductCardProps> = ({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  autoOpenPlanModal = false,
}) => {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();
  const [isPlanModalOpen, setPlanModalOpen] = useState(autoOpenPlanModal);
  const [isCancelOpen, setCancelOpen] = useState(false);

  const { data: subscription, isLoading: subscriptionLoading } =
    trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    { staleTime: 60_000 },
  );

  const currentPlan = subscription?.plan ?? null;

  // Usage is only worth fetching (and rendering) once there is a plan.
  const { data: usage } = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });
  const { data: invoice } = trpc.stripe.getUpcomingInvoice.useQuery(undefined, {
    enabled: Boolean(currentPlan),
    staleTime: 30_000,
  });

  const revalidate = async () => {
    await Promise.all([
      trpcUtils.stripe.getDeploySubscription.invalidate(),
      trpcUtils.stripe.getUpcomingInvoice.invalidate(),
      trpcUtils.workspace.getCurrent.invalidate(),
      trpcUtils.stripe.getDeploySubscription.refetch(),
    ]);
  };

  const subscribe = trpc.stripe.subscribeDeploy.useMutation({
    onSuccess: async () => {
      setPlanModalOpen(false);
      toast.success("Subscribed to Compute");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async () => {
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
    return <div className="h-[150px] w-full animate-pulse rounded-xl bg-grayA-3" />;
  }

  // Deploy billing not configured server-side: hide the card entirely.
  if (plansData && !plansData.configured) {
    return null;
  }

  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);

  // Credits equal the plan fee; usage beyond them is billed on top.
  const credits = currentPlanOption?.amount ?? null;
  const usageAmount = invoice?.deployUsageAmount ?? null;

  const meterStats = usage
    ? [
        { label: "CPU", value: `${formatQuantity(usage.cpuSeconds / 3600)} hrs` },
        { label: "Memory", value: `${formatQuantity(usage.memoryGiBHours)} GiB-hrs` },
        { label: "Egress", value: `${formatQuantity(usage.egressGiB)} GiB` },
        { label: "Disk", value: `${formatQuantity(usage.diskGiBHours)} GiB-hrs` },
      ]
    : null;

  return (
    <>
      <ProductCard
        icon={<Cube iconSize="md-regular" />}
        iconClassName="bg-orangeA-3 text-orange-11"
        name="Compute"
        tag={currentPlan ? (currentPlanOption?.name ?? currentPlan) : undefined}
        subtitle={
          currentPlan
            ? credits !== null
              ? `${formatDollars(credits)}/${currentPlanOption?.interval ?? "month"}, includes ${formatDollars(credits)} of usage credits`
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
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <Button
                  variant="primary"
                  size="md"
                  disabled={!isAdmin}
                  onClick={() => {
                    // No card yet: go through checkout first; /success hands
                    // the intent back and this picker reopens automatically.
                    if (hasPaymentMethod) {
                      setPlanModalOpen(true);
                    } else {
                      router.push(
                        `/${workspaceSlug}/settings/billing/stripe/checkout?intent=compute`,
                      );
                    }
                  }}
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
            <UsageMeter
              label="Usage this period"
              value={
                usageAmount !== null && credits !== null
                  ? `${formatDollars(usageAmount)} of ${formatDollars(credits)} credits`
                  : usageAmount !== null
                    ? formatDollars(usageAmount)
                    : "—"
              }
              fraction={usageAmount !== null && credits ? usageAmount / credits : null}
              fillClassName="bg-orange-9"
            />
            {meterStats ? (
              <div className="grid grid-cols-2 gap-px overflow-hidden rounded-lg bg-grayA-3 sm:grid-cols-4">
                {meterStats.map((stat) => (
                  <div key={stat.label} className="bg-white px-3 py-2 dark:bg-black">
                    <p className="text-[11px] text-gray-10 uppercase tracking-wide">{stat.label}</p>
                    <p className="font-medium text-[13px] text-gray-12 tabular-nums">
                      {stat.value}
                    </p>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        ) : null}
      </ProductCard>

      <PlanChangeModal
        isOpen={isPlanModalOpen}
        onOpenChange={setPlanModalOpen}
        title={currentPlan ? "Change Compute plan" : "Choose a Compute plan"}
        subTitle="The monthly plan fee includes the same amount of usage credits; usage beyond them is billed on top."
        options={plans.map((plan) => ({
          id: plan.plan,
          name: plan.name,
          amount: plan.amount,
          interval: plan.interval,
          detail:
            plan.amount !== null
              ? `${formatDollars(plan.amount)} usage credits included`
              : (plan.description ?? "Custom pricing and credits"),
        }))}
        currentId={currentPlan}
        // Warn when the period's metered spend already exceeds the credits the
        // selected plan includes: the downgrade is allowed, but the difference
        // becomes overage instead of being covered.
        warningFor={(option) =>
          option.amount !== null && usageAmount !== null && usageAmount > option.amount
            ? `Your usage this period (${formatDollars(usageAmount)}) already exceeds the ${formatDollars(
                option.amount,
              )} of credits ${option.name} includes; the difference is billed as overage.`
            : null
        }
        changeNote="Takes effect immediately; the fee difference is prorated on your next invoice."
        submittingId={
          subscribe.isLoading
            ? subscribe.variables?.plan
            : change.isLoading
              ? change.variables?.plan
              : undefined
        }
        onSelect={(id) => {
          // The option ids are plan names by construction; find() narrows the
          // string back to DeployPlan without a cast.
          const plan = DEPLOY_PLANS.find((p) => p === id);
          if (!plan) {
            return;
          }
          if (currentPlan) {
            change.mutate({ plan });
          } else {
            subscribe.mutate({ plan });
          }
        }}
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
