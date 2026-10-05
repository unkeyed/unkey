"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import type Stripe from "stripe";
import { ApiAddOnCard } from "./components/api-addon-card";
import { BillingSummary } from "./components/billing-summary";
import { DeployProductCard } from "./components/deploy-product-card";
import { SubscriptionStatus } from "./components/subscription-status";

function Shell({ children }: { children: ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Billing</PageHeaderTitle>
          <PageHeaderDescription>
            Manage your plans, usage, and payment methods.
          </PageHeaderDescription>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            variant="outline"
            size="md"
            render={
              <Link
                href="https://cal.com/james-r-perkins/sales"
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            Schedule a call
          </Button>
          <Button variant="primary" size="md" render={<Link href="mailto:support@unkey.com" />}>
            Contact us
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}

/**
 * Billing page shown when the deployBilling flag is on: a flat list of
 * product cards, Compute first (hero with spend against credits), API
 * management below it. The headline strip shows the billing period and
 * upcoming invoice, Vercel-style. Cancelling lives as a quiet link inside
 * each product card, so there is no danger zone competing with the upgrade
 * actions. Flag-off keeps the existing single-product page (./client).
 */
export const DeployBillingClient: React.FC = () => {
  const workspace = useWorkspaceNavigation();

  // Server-side `requireWorkspaceAdmin` enforces this on every billing
  // mutation; we mirror it on the client purely for UX so non-admin members
  // get a clear "admin required" affordance instead of a request that fails
  // with FORBIDDEN.
  const { user: currentUser } = useWorkspace();
  const isAdmin = currentUser?.role === "admin";

  const {
    data: billingInfo,
    isLoading: billingLoading,
    error: billingError,
  } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  const subscription = billingInfo?.subscription;
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);

  return (
    <Shell>
      <div className="flex w-full flex-col gap-4 pt-4 pb-16">
        {subscription ? (
          <SubscriptionStatus status={subscription.status as Stripe.Subscription.Status} />
        ) : null}

        <BillingSummary
          workspaceSlug={workspace.slug}
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
        />

        <DeployProductCard isAdmin={isAdmin} hasPaymentMethod={hasPaymentMethod} />

        {billingError ? (
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>Failed to load API billing information</EmptyStateTitle>
              <EmptyStateDescription>
                There was an error loading your API billing information. Please try again later.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        ) : billingLoading || !billingInfo ? (
          <Skeleton className="h-[120px] w-full rounded-lg" />
        ) : (
          <ApiAddOnCard
            isAdmin={isAdmin}
            products={billingInfo.products}
            subscription={subscription}
            currentProductId={billingInfo.currentProductId}
          />
        )}
      </div>
    </Shell>
  );
};
