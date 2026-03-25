"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, SettingCard, SettingCardGroup, SettingsDangerZone } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import type Stripe from "stripe";
import { WorkspaceNavbar } from "../workspace-navbar";
import { CancelAlert } from "./components/cancel-alert";
import { CancelPlan } from "./components/cancel-plan";
import { CurrentPlanCard } from "./components/current-plan-card";
import { FreeTierAlert } from "./components/free-tier-alert";
import { PlanSelectionModal } from "./components/plan-selection-modal";
import { Shell } from "./components/shell";
import { SubscriptionStatus } from "./components/subscription-status";
import { Usage } from "./components/usage";

const MAX_QUOTA = 150000;

export const Client: React.FC = () => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const [showPlanModal, setShowPlanModal] = useState(false);

  // Fetch billing info using new tRPC route
  // Query is automatically keyed by workspace context and will refetch on workspace change
  const {
    data: billingInfo,
    isLoading: billingLoading,
    error: billingError,
  } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    // Cache for 30 seconds to reduce unnecessary refetches
    // TRPC automatically scopes by workspace via requireWorkspace middleware
    staleTime: 30_000, // 30 seconds
  });

  // Handle loading states - don't render until we have billing info
  if (billingLoading || !billingInfo) {
    return (
      <div className="animate-pulse">
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <Shell>
          <div className="flex flex-col gap-2 items-center">
            <span className="font-semibold text-gray-12 leading-8 text-lg">Billing</span>
            <span className="leading-4 text-gray-11 text-[13px]">
              Manage your subscription, usage, and payment methods.
            </span>
          </div>
          <div className="w-full h-[150px] bg-grayA-3 rounded-lg mt-1" />
          <div className="w-full h-[90px] bg-grayA-3 rounded-lg" />
          <div className="w-full h-[90px] bg-grayA-3 rounded-lg" />
        </Shell>
      </div>
    );
  }

  // Handle error states
  if (billingError) {
    return (
      <div>
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <Empty>
          <Empty.Title>Failed to load billing information</Empty.Title>
          <Empty.Description>
            There was an error loading your billing information. Please try again later.
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  // Extract data from tRPC responses
  const products = billingInfo.products;
  const subscription = billingInfo.subscription;
  const currentProductId = billingInfo.currentProductId;

  const hasPaidSubscription = Boolean(
    subscription &&
      currentProductId &&
      ["active", "trialing", "past_due"].includes(subscription.status),
  );
  const isFreeTier = !hasPaidSubscription;
  const allowCancel = subscription && subscription.status === "active" && !subscription.cancelAt;
  const currentProduct = hasPaidSubscription
    ? products.find((p) => p.id === currentProductId)
    : undefined;

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
      <Shell>
        {subscription ? (
          <SubscriptionStatus
            workspaceSlug={workspace.slug}
            status={subscription.status as Stripe.Subscription.Status}
          />
        ) : null}
        <div className="flex flex-col gap-2 items-center">
          <span className="font-semibold text-gray-12 leading-8 text-lg">Billing</span>
          <span className="leading-4 text-gray-11 text-[13px]">
            Manage your subscription, usage, and payment methods.
          </span>
        </div>

        {isFreeTier ? <FreeTierAlert /> : null}

        {workspace.stripeCustomerId ? (
          <div className="w-full">
            <SettingCardGroup>
              <CurrentPlanCard
                currentProduct={currentProduct}
                onChangePlan={() => setShowPlanModal(true)}
              />
              <Usage quota={currentProduct?.quotas?.requestsPerMonth ?? MAX_QUOTA} />
              <SettingCard
                title="Billing Portal"
                description="Manage payment methods and see your invoices."
              >
                <div className="w-full flex h-full items-center justify-end gap-4">
                  <Button
                    variant="outline"
                    className="py-2 px-3 text-gray-12 font-medium text-sm bg-grayA-2 hover:bg-grayA-3"
                    aria-label="Open billing portal"
                    onClick={() => {
                      router.push(`/${workspace.slug}/settings/billing/stripe/portal`);
                    }}
                  >
                    Open Portal
                  </Button>
                </div>
              </SettingCard>
            </SettingCardGroup>

            <PlanSelectionModal
              isOpen={showPlanModal}
              onOpenChange={setShowPlanModal}
              products={products}
              currentProductId={currentProductId}
              workspaceSlug={workspace.slug}
              isChangingPlan={Boolean(subscription)}
            />
          </div>
        ) : (
          <div className="w-full">
            <SettingCardGroup>
              <SettingCard
                title="Add payment method"
                description="Before upgrading, you need to add a payment method."
                contentWidth="w-full lg:w-[320px]"
              >
                <div className="flex justify-end w-full">
                  <Button
                    variant="outline"
                    className="px-3 py-2 text-gray-12 font-medium text-[13px] bg-grayA-2 shadow-md hover:bg-grayA-3"
                    aria-label="Add payment method"
                    onClick={() => {
                      router.push(`/${workspace.slug}/settings/billing/stripe/checkout`);
                    }}
                  >
                    Add payment method
                  </Button>
                </div>
              </SettingCard>
              <Usage quota={currentProduct?.quotas?.requestsPerMonth ?? MAX_QUOTA} />
            </SettingCardGroup>
          </div>
        )}

        <CancelAlert cancelAt={subscription?.cancelAt} />

        {allowCancel ? (
          <SettingsDangerZone>
            <CancelPlan />
          </SettingsDangerZone>
        ) : null}
      </Shell>
    </div>
  );
};
