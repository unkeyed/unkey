"use client";

import { formatPrice } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { useEffect, useState } from "react";
import { BillingCard, BillingCardGroup, BillingSection, billingButton } from "./billing-card";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";
const NEEDS_PAYMENT_TOOLTIP = "Add a payment method before subscribing to a Compute plan";

type DeployBillingSectionProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
};

/**
 * Flag-gated Unkey Deploy billing section. Reads the current plan from the local
 * deploy_plan signal (getDeploySubscription) and the available tiers from Stripe
 * (getDeployPlans). Subscribe / change / cancel mutate Stripe; the webhook syncs
 * deploy_plan, so on success we just refetch.
 */
export const DeployBillingSection: React.FC<DeployBillingSectionProps> = ({
  isAdmin,
  hasPaymentMethod,
}) => {
  const trpcUtils = trpc.useUtils();
  const [isPlanModalOpen, setPlanModalOpen] = useState(false);

  const { data: subscription, isLoading: subscriptionLoading } =
    trpc.stripe.getDeploySubscription.useQuery(undefined, { staleTime: 30_000 });
  const { data: plansData, isLoading: plansLoading } = trpc.stripe.getDeployPlans.useQuery(
    undefined,
    { staleTime: 60_000 },
  );

  const revalidate = async () => {
    await Promise.all([
      trpcUtils.stripe.getDeploySubscription.invalidate(),
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
  if (subscriptionLoading || plansLoading) {
    return <div className="w-full h-[120px] bg-grayA-3 animate-pulse" />;
  }

  // Deploy billing not configured server-side: hide the section entirely.
  if (plansData && !plansData.configured) {
    return null;
  }

  const currentPlan = subscription?.plan ?? null;
  const plans = plansData?.plans ?? [];
  const currentPlanOption = plans.find((p) => p.plan === currentPlan);
  const manageDisabled = !isAdmin;
  const subscribeDisabled = !isAdmin || !hasPaymentMethod;
  const subscribeTooltip = isAdmin
    ? hasPaymentMethod
      ? undefined
      : NEEDS_PAYMENT_TOOLTIP
    : ADMIN_ONLY_TOOLTIP;

  return (
    <BillingSection label="Compute">
      <BillingCardGroup>
        <BillingCard
          label="Current plan"
          title={
            currentPlan ? (
              <span className="font-medium text-base text-gray-12 tracking-tight">
                {currentPlanOption?.name ?? currentPlan}
              </span>
            ) : undefined
          }
          description={
            currentPlan ? (
              currentPlanOption?.amount != null ? (
                <span>
                  <span className="font-mono">{formatPrice(currentPlanOption.amount)}/mo</span> +
                  usage
                </span>
              ) : (
                "Custom pricing. Usage is billed on top of the plan fee."
              )
            ) : (
              "Subscribe to a Compute plan to start deploying projects."
            )
          }
        >
          <InfoTooltip
            content={currentPlan ? ADMIN_ONLY_TOOLTIP : subscribeTooltip}
            disabled={currentPlan ? !manageDisabled : !subscribeDisabled}
            asChild
          >
            <span>
              <Button
                variant={currentPlan ? "outline" : "primary"}
                size="lg"
                className={billingButton}
                disabled={currentPlan ? manageDisabled : subscribeDisabled}
                onClick={() => setPlanModalOpen(true)}
              >
                {currentPlan ? "Change plan" : "Choose a plan"}
              </Button>
            </span>
          </InfoTooltip>
        </BillingCard>
      </BillingCardGroup>

      <DeployPlanModal
        isOpen={isPlanModalOpen}
        onOpenChange={setPlanModalOpen}
        plans={plans}
        currentPlan={currentPlan}
        isSubmittingPlan={subscribe.isLoading ? subscribe.variables?.plan : change.variables?.plan}
        onSelect={(plan) => {
          if (currentPlan) {
            change.mutate({ plan });
          } else {
            subscribe.mutate({ plan });
          }
        }}
      />
    </BillingSection>
  );
};

type DeployPlanOption = {
  plan: DeployPlan;
  name: string;
  description: string | null;
  amount: number | null;
  currency: string;
  interval: string | null;
};

type DeployPlanModalProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  plans: DeployPlanOption[];
  currentPlan: DeployPlan | null;
  /** The plan whose mutation is in flight, for the per-button loading state. */
  isSubmittingPlan: DeployPlan | undefined;
  onSelect: (plan: DeployPlan) => void;
};

const DeployPlanModal: React.FC<DeployPlanModalProps> = ({
  isOpen,
  onOpenChange,
  plans,
  currentPlan,
  isSubmittingPlan,
  onSelect,
}) => {
  const [selected, setSelected] = useState<DeployPlan | null>(currentPlan);

  // Reset the highlighted plan to the current one whenever the modal opens.
  useEffect(() => {
    if (isOpen) {
      setSelected(currentPlan);
    }
  }, [isOpen, currentPlan]);

  const isSubmitting = isSubmittingPlan !== undefined;
  const ctaDisabled = !selected || selected === currentPlan || isSubmitting;

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title={currentPlan ? "Change Compute plan" : "Choose a Compute plan"}
      subTitle="The plan fee is billed monthly; usage is billed on top."
      className="rounded-none!"
      footer={
        <Button
          type="button"
          variant="primary"
          size="xlg"
          className={cn("w-full", billingButton)}
          loading={isSubmitting}
          disabled={ctaDisabled}
          onClick={() => {
            if (selected) {
              onSelect(selected);
            }
          }}
        >
          {currentPlan ? "Change plan" : "Subscribe"}
        </Button>
      }
    >
      <div className="grid grid-cols-1 border-t border-l border-grayA-4 sm:grid-cols-3">
        {plans.map((plan) => {
          const isCurrent = plan.plan === currentPlan;
          const isSelected = plan.plan === selected;
          return (
            <button
              type="button"
              key={plan.plan}
              onClick={() => setSelected(plan.plan)}
              className={cn(
                "relative flex min-w-0 flex-col gap-5 border-r border-b border-grayA-4 px-5 pt-5 pb-6 text-left transition-colors",
                "outline-none focus-visible:bg-grayA-2",
                isSelected ? "bg-grayA-2 ring-1 ring-inset ring-gray-12" : "hover:bg-grayA-2",
              )}
            >
              <div className="flex items-center justify-between">
                <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-gray-11">
                  {plan.name}
                </span>
                {isCurrent ? (
                  <span className="font-mono text-[10px] uppercase tracking-wider text-gray-9">
                    Current
                  </span>
                ) : null}
              </div>
              <div className="flex items-baseline gap-1">
                {plan.amount !== null ? (
                  <>
                    <span className="text-3xl font-medium tracking-tight text-gray-12">
                      {formatPrice(plan.amount)}
                    </span>
                    {plan.interval ? (
                      <span className="text-gray-9 text-sm">
                        /{plan.interval === "month" ? "mo" : plan.interval}
                      </span>
                    ) : null}
                  </>
                ) : (
                  <span className="font-medium text-gray-12 text-xl tracking-tight">
                    Contact us
                  </span>
                )}
              </div>
              {plan.description ? (
                <p className="text-[13px] text-gray-10 leading-snug">{plan.description}</p>
              ) : null}
            </button>
          );
        })}
      </div>
    </DialogContainer>
  );
};
