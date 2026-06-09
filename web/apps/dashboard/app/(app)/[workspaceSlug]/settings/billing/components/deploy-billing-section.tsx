"use client";

import { formatPrice } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import {
  Button,
  DialogContainer,
  InfoTooltip,
  SettingCard,
  SettingCardGroup,
  SettingsDangerZone,
  SettingsZoneRow,
  toast,
} from "@unkey/ui";
import { useEffect, useState } from "react";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";
const NEEDS_PAYMENT_TOOLTIP = "Add a payment method before subscribing to Unkey Deploy";

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
  const [isCancelOpen, setCancelOpen] = useState(false);

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
      toast.success("Subscribed to Unkey Deploy");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async () => {
      setPlanModalOpen(false);
      toast.success("Unkey Deploy plan changed");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const cancel = trpc.stripe.cancelDeploy.useMutation({
    onSuccess: async () => {
      setCancelOpen(false);
      toast.info("Unkey Deploy cancelled");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  if (subscriptionLoading || plansLoading) {
    return <div className="w-full h-[120px] bg-grayA-3 rounded-lg animate-pulse" />;
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
    <div className="w-full">
      <SettingCardGroup>
        <SettingCard
          title="Unkey Deploy"
          description={
            currentPlan
              ? `You are on the ${currentPlanOption?.name ?? currentPlan} plan. Usage is billed on top of the plan fee.`
              : "Deploy requires a plan. Subscribe to start deploying projects."
          }
        >
          <div className="w-full flex h-full items-center justify-end gap-4">
            <InfoTooltip
              content={currentPlan ? ADMIN_ONLY_TOOLTIP : subscribeTooltip}
              disabled={currentPlan ? !manageDisabled : !subscribeDisabled}
              asChild
            >
              <span>
                <Button
                  variant={currentPlan ? "outline" : "primary"}
                  className="py-2 px-3 font-medium text-sm"
                  disabled={currentPlan ? manageDisabled : subscribeDisabled}
                  onClick={() => setPlanModalOpen(true)}
                >
                  {currentPlan ? "Change plan" : "Choose a plan"}
                </Button>
              </span>
            </InfoTooltip>
          </div>
        </SettingCard>
      </SettingCardGroup>

      {currentPlan ? (
        <SettingsDangerZone>
          <SettingsZoneRow
            title="Cancel Unkey Deploy"
            description={
              manageDisabled
                ? ADMIN_ONLY_TOOLTIP
                : "Stops Unkey Deploy immediately and removes it from your subscription. Usage so far is billed; the plan fee is not refunded."
            }
            action={{
              label: "Cancel Deploy",
              onClick: () => setCancelOpen(true),
              disabled: manageDisabled,
            }}
          />
        </SettingsDangerZone>
      ) : null}

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

      <DialogContainer
        isOpen={isCancelOpen}
        onOpenChange={setCancelOpen}
        title="Cancel Unkey Deploy"
        subTitle="Stop deploying with this workspace"
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
            Cancel Unkey Deploy
          </Button>
        }
      >
        <div className="text-gray-11 text-[13px] leading-6">
          Cancelling stops Unkey Deploy immediately: your deployments stop and no further usage is
          billed. Usage up to now is still charged, and the plan fee already paid is not refunded.
        </div>
      </DialogContainer>
    </div>
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
      title={currentPlan ? "Change Unkey Deploy plan" : "Choose an Unkey Deploy plan"}
      subTitle="The plan fee is billed monthly; usage is billed on top."
      footer={
        <Button
          type="button"
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
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
      <div className="flex flex-col gap-3">
        {plans.map((plan) => {
          const isCurrent = plan.plan === currentPlan;
          const isSelected = plan.plan === selected;
          return (
            <button
              type="button"
              key={plan.plan}
              onClick={() => setSelected(plan.plan)}
              className={cn(
                "w-full text-left rounded-lg border px-4 py-3 transition-colors",
                isSelected ? "border-accent-9 bg-accentA-2" : "border-grayA-4 hover:bg-grayA-2",
              )}
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-gray-12 text-sm">{plan.name}</span>
                <span className="text-gray-12 text-sm">
                  {plan.amount !== null ? (
                    <>
                      {formatPrice(plan.amount)}
                      {plan.interval ? `/${plan.interval}` : ""}
                    </>
                  ) : (
                    "Contact us"
                  )}
                </span>
              </div>
              {plan.description ? (
                <p className="text-gray-10 text-[13px] mt-1">{plan.description}</p>
              ) : null}
              {isCurrent ? <p className="text-gray-9 text-xs mt-1">Current plan</p> : null}
            </button>
          );
        })}
      </div>
    </DialogContainer>
  );
};
