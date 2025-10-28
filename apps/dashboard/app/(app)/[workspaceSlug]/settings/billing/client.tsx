"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, SettingCard } from "@unkey/ui";
import Link from "next/link";
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
  const workspace = useWorkspaceNavigation();
  const [showPlanModal, setShowPlanModal] = useState(false);

  // Fetch billing info using new tRPC route
  const {
    data: billingInfo,
    isLoading: billingLoading,
    error: billingError,
  } = trpc.stripe.getBillingInfo.useQuery();

  // Handle loading states
  if (billingLoading) {
    return (
      <div className="animate-pulse">
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <Shell>
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
            There was an error loading your billing information. Please try
            again later.
          </Empty.Description>
        </Empty>
      </div>
    );
  }

  // Extract data from tRPC responses
  const products = billingInfo?.products ?? [];
  const subscription = billingInfo?.subscription;
  const currentProductId = billingInfo?.currentProductId;

  const allowUpdate =
    subscription && ["active", "trialing"].includes(subscription.status);

  const isFreeTier = !subscription || subscription.status !== "active";
  const allowCancel =
    subscription && subscription.status === "active" && !subscription.cancelAt;
  const currentProduct = allowUpdate
    ? products.find((p) => p.id === currentProductId)
    : undefined;

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
      <Shell>
        {subscription ? (
          <SubscriptionStatus
            workspaceId={workspace.id}
            workspaceSlug={workspace.slug}
            status={subscription.status as Stripe.Subscription.Status}
          />
        ) : null}

        <CancelAlert cancelAt={subscription?.cancelAt} />
        {isFreeTier ? <FreeTierAlert /> : null}
        <Usage quota={currentProduct?.quotas?.requestsPerMonth || MAX_QUOTA} />

        {workspace.stripeCustomerId ? (
          <>
            <CurrentPlanCard
              currentProduct={currentProduct}
              onChangePlan={() => setShowPlanModal(true)}
            />

            <PlanSelectionModal
              isOpen={showPlanModal}
              onOpenChange={setShowPlanModal}
              products={products}
              workspaceSlug={workspace.slug}
              currentProductId={currentProductId}
              isChangingPlan={true}
            />
          </>
        ) : (
          <SettingCard
            title="Add payment method"
            border="both"
            description="Before upgrading, you need to add a payment method."
            className="sm:w-full text-wrap w-full"
            contentWidth="w-full"
          >
            <div className="flex justify-end w-full">
              <Button variant="primary">
                <Link
                  href={`/${workspace.slug}/settings/billing/stripe/checkout`}
                >
                  Add payment method
                </Link>
              </Button>
            </div>
          </SettingCard>
        )}
        {workspace.stripeCustomerId ? (
          <SettingCard
            title="Billing Portal"
            border="both"
            description="Manage Payment methods and see your invoices."
            className="w-full"
            contentWidth="w-full lg:w-[320px]"
          >
            <div className="w-full flex h-full items-center justify-end gap-4">
              <Button variant="outline" size="lg">
                <Link
                  href={`/${workspace.slug}/settings/billing/stripe/portal`}
                >
                  Open Portal
                </Link>
              </Button>
            </div>
          </SettingCard>
        ) : null}

        {subscription && allowCancel ? <CancelPlan /> : null}
      </Shell>
    </div>
  );
};
