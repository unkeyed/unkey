"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
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
          <Button asChild variant="outline" size="md">
            <Link
              href="https://cal.com/james-r-perkins/sales"
              target="_blank"
              rel="noopener noreferrer"
            >
              Schedule a call
            </Link>
          </Button>
          <Button asChild variant="primary" size="md">
            <Link href="mailto:support@unkey.com">Contact us</Link>
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
  const router = useRouter();
  const searchParams = useSearchParams();

  // Post-checkout hand-off: /success sends the user back here with the
  // intent their checkout round-trip started with, and we reopen the plan
  // picker they were heading for. Captured once and stripped from the URL so
  // refreshes don't reopen modals.
  const [checkoutIntent] = useState(() => searchParams?.get("intent") ?? null);
  useEffect(() => {
    if (searchParams?.get("intent")) {
      router.replace(`/${workspace.slug}/settings/billing`);
    }
  }, [searchParams, router, workspace.slug]);

  // Server-side `requireWorkspaceAdmin` enforces this on every billing
  // mutation; we mirror it on the client purely for UX so non-admin members
  // get a clear "admin required" affordance instead of a request that fails
  // with FORBIDDEN.
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser?.role === "admin";

  const {
    data: billingInfo,
    isLoading: billingLoading,
    error: billingError,
  } = trpc.stripe.getBillingInfo.useQuery(undefined, { staleTime: 30_000 });

  if (billingLoading || !billingInfo) {
    return (
      <Shell>
        <div className="animate-pulse">
          <div className="flex w-full flex-col items-center gap-4 pt-4 pb-16">
            <div className="h-[72px] w-full rounded-xl bg-grayA-3" />
            <div className="h-[180px] w-full rounded-xl bg-grayA-3" />
            <div className="h-[120px] w-full rounded-xl bg-grayA-3" />
          </div>
        </div>
      </Shell>
    );
  }

  if (billingError) {
    return (
      <Shell>
        <Empty>
          <Empty.Title>Failed to load billing information</Empty.Title>
          <Empty.Description>
            There was an error loading your billing information. Please try again later.
          </Empty.Description>
        </Empty>
      </Shell>
    );
  }

  const subscription = billingInfo.subscription;
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);

  return (
    <Shell>
      <div className="flex w-full flex-col gap-4 pt-4 pb-16">
        {subscription ? (
          <SubscriptionStatus
            workspaceSlug={workspace.slug}
            status={subscription.status as Stripe.Subscription.Status}
          />
        ) : null}

        <BillingSummary
          workspaceSlug={workspace.slug}
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
        />

        <DeployProductCard
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
          workspaceSlug={workspace.slug}
          autoOpenPlanModal={checkoutIntent === "compute" && hasPaymentMethod}
        />

        <ApiAddOnCard
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
          workspaceSlug={workspace.slug}
          products={billingInfo.products}
          subscription={subscription}
          currentProductId={billingInfo.currentProductId}
          autoOpenPlanModal={checkoutIntent === "api" && hasPaymentMethod}
        />
      </div>
    </Shell>
  );
};
