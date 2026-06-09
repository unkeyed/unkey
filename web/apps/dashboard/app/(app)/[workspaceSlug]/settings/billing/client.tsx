"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useFlag } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, InfoTooltip, SettingsShell } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import type Stripe from "stripe";
import { WorkspaceNavbar } from "../workspace-navbar";
import {
  BillingCard,
  BillingCardGroup,
  BillingSection,
  billingButton,
} from "./components/billing-card";
import { CancelAlert } from "./components/cancel-alert";
import { CancelCompute } from "./components/cancel-compute";
import { CancelPlan } from "./components/cancel-plan";
import { CurrentPlanCard } from "./components/current-plan-card";
import { DeployBillingSection } from "./components/deploy-billing-section";
import { FreeTierAlert } from "./components/free-tier-alert";
import { PlanSelectionModal } from "./components/plan-selection-modal";
import { SubscriptionStatus } from "./components/subscription-status";
import { Usage } from "./components/usage";

const MAX_QUOTA = 150000;

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

export const Client: React.FC = () => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const [showPlanModal, setShowPlanModal] = useState(false);
  const deployBillingEnabled = useFlag("deployBilling");

  // Server-side `requireWorkspaceAdmin` enforces this on every billing
  // mutation; we mirror it on the client purely for UX so non-admin members
  // get a clear "admin required" affordance instead of a request that fails
  // with FORBIDDEN.
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser?.role === "admin";

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

  // The Compute cancellation lives in the shared danger zone below, so the
  // page needs to know whether a Compute plan is active. React-query dedupes
  // this with the identical query inside DeployBillingSection.
  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
    enabled: deployBillingEnabled,
  });
  const hasComputePlan = deployBillingEnabled && Boolean(deploySubscription?.plan);

  // Handle loading states - don't render until we have billing info
  if (billingLoading || !billingInfo) {
    return (
      <div className="animate-pulse">
        <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
        <SettingsShell>
          <div className="flex w-full flex-col gap-1.5">
            <span className="font-medium text-gray-12 text-lg tracking-tight">Billing</span>
            <span className="text-[13px] text-gray-11 leading-5">
              Manage your subscription, usage, and payment methods.
            </span>
          </div>
          <div className="w-full h-[150px] bg-grayA-3 mt-1" />
          <div className="w-full h-[90px] bg-grayA-3" />
          <div className="w-full h-[90px] bg-grayA-3" />
        </SettingsShell>
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
      <SettingsShell>
        {subscription ? (
          <SubscriptionStatus
            workspaceSlug={workspace.slug}
            status={subscription.status as Stripe.Subscription.Status}
          />
        ) : null}
        <div className="flex w-full flex-col gap-1.5">
          <span className="font-medium text-gray-12 text-lg tracking-tight">Billing</span>
          <span className="text-[13px] text-gray-11 leading-5">
            Manage your subscription, usage, and payment methods.
          </span>
        </div>

        {isFreeTier ? <FreeTierAlert /> : null}

        {workspace.stripeCustomerId ? (
          <BillingSection label="API plan">
            <BillingCardGroup>
              <CurrentPlanCard
                currentProduct={currentProduct}
                onChangePlan={() => setShowPlanModal(true)}
                disabled={!isAdmin}
                disabledReason={ADMIN_ONLY_TOOLTIP}
              />
              <Usage quota={currentProduct?.quotas?.requestsPerMonth ?? MAX_QUOTA} />
            </BillingCardGroup>

            <PlanSelectionModal
              isOpen={showPlanModal}
              onOpenChange={setShowPlanModal}
              products={products}
              currentProductId={currentProductId}
              workspaceSlug={workspace.slug}
              isChangingPlan={Boolean(subscription)}
            />
          </BillingSection>
        ) : (
          <BillingSection label="API plan">
            <BillingCardGroup>
              <BillingCard
                label="Payment method"
                title="Add a payment method"
                description="Before upgrading, you need to add a payment method."
              >
                <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                  <span>
                    <Button
                      variant="outline"
                      size="lg"
                      className={billingButton}
                      aria-label="Add payment method"
                      disabled={!isAdmin}
                      onClick={() => {
                        router.push(`/${workspace.slug}/settings/billing/stripe/checkout`);
                      }}
                    >
                      Add payment method
                    </Button>
                  </span>
                </InfoTooltip>
              </BillingCard>
              <Usage quota={currentProduct?.quotas?.requestsPerMonth ?? MAX_QUOTA} />
            </BillingCardGroup>
          </BillingSection>
        )}

        {deployBillingEnabled ? (
          <DeployBillingSection
            isAdmin={isAdmin}
            hasPaymentMethod={Boolean(workspace.stripeCustomerId)}
          />
        ) : null}

        {workspace.stripeCustomerId ? (
          <BillingSection label="Billing portal">
            <BillingCardGroup>
              <BillingCard
                title="Payment methods and invoices"
                description="Opens the Stripe billing portal. Covers every subscription in this workspace."
              >
                <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                  <span>
                    <Button
                      variant="outline"
                      size="lg"
                      className={billingButton}
                      aria-label="Open billing portal"
                      disabled={!isAdmin}
                      onClick={() => {
                        router.push(`/${workspace.slug}/settings/billing/stripe/portal`);
                      }}
                    >
                      Open Portal
                    </Button>
                  </span>
                </InfoTooltip>
              </BillingCard>
            </BillingCardGroup>
          </BillingSection>
        ) : null}

        <CancelAlert
          cancelAt={subscription?.cancelAt}
          disabled={!isAdmin}
          disabledReason={ADMIN_ONLY_TOOLTIP}
        />

        {allowCancel || hasComputePlan ? (
          <BillingSection label="Danger zone">
            <BillingCardGroup className="border-error-7 divide-error-7">
              {hasComputePlan ? (
                <CancelCompute disabled={!isAdmin} disabledReason={ADMIN_ONLY_TOOLTIP} />
              ) : null}
              {allowCancel ? (
                <CancelPlan disabled={!isAdmin} disabledReason={ADMIN_ONLY_TOOLTIP} />
              ) : null}
            </BillingCardGroup>
          </BillingSection>
        ) : null}
      </SettingsShell>
    </div>
  );
};
