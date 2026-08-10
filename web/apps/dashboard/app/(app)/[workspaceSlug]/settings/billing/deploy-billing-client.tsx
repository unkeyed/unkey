"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
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
import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import type Stripe from "stripe";
import { ApiAddOnCard } from "./components/api-addon-card";
import { BillingSummary } from "./components/billing-summary";
import { CostControl } from "./components/cost-control";
import { DeployProductCard } from "./components/deploy-product-card";
import { RelatedPages } from "./components/related-pages";
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

function SectionLabel({ children }: { children: ReactNode }) {
  return <h2 className="pt-2 font-medium text-[13px] text-gray-12">{children}</h2>;
}

/**
 * Billing page shown when the deployBilling flag is on: the upcoming invoice,
 * then the plan per product, then cost control split per product, then the pages
 * that answer the questions this one deliberately does not — Usage for
 * consumption, Limits for the ceilings we impose. The spend budget lives under
 * cost control rather than inside the Compute plan card, because a cap is a
 * billing preference and not part of the plan. Flag-off keeps the existing
 * single-product page (./client).
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
  // Strip the ?intent= param once, after capturing it above. Guarded with a ref
  // because router.replace is async: searchParams stays populated until the
  // navigation commits, so a re-render in that window (router/slug identity
  // change) would otherwise re-fire the replace.
  const intentCleared = useRef(false);
  useEffect(() => {
    if (checkoutIntent && !intentCleared.current) {
      intentCleared.current = true;
      router.replace(routes.settings.billing({ workspaceSlug: workspace.slug }));
    }
  }, [checkoutIntent, router, workspace.slug]);

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
  } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  const subscription = billingInfo?.subscription;
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);

  // Cost control needs the same plan, catalog and usage reads the Compute card
  // makes, so these resolve from cache rather than adding round-trips.
  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });
  const { data: deployPlans } = trpc.stripe.getDeployPlans.useQuery(undefined, {
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });
  const { data: deployUsage } = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: Boolean(deploySubscription?.plan),
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

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

        <SectionLabel>Plans</SectionLabel>

        <DeployProductCard
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
          workspaceSlug={workspace.slug}
          autoOpenPlanModal={checkoutIntent === "compute" && hasPaymentMethod}
        />

        {billingError ? (
          <Empty>
            <Empty.Title>Failed to load API billing information</Empty.Title>
            <Empty.Description>
              There was an error loading your API billing information. Please try again later.
            </Empty.Description>
          </Empty>
        ) : billingLoading || !billingInfo ? (
          <div className="h-[120px] w-full animate-pulse rounded-lg bg-grayA-3" />
        ) : (
          <ApiAddOnCard
            isAdmin={isAdmin}
            hasPaymentMethod={hasPaymentMethod}
            workspaceSlug={workspace.slug}
            products={billingInfo.products}
            subscription={subscription}
            currentProductId={billingInfo.currentProductId}
            autoOpenPlanModal={checkoutIntent === "api" && hasPaymentMethod}
          />
        )}

        <SectionLabel>Cost control</SectionLabel>

        <CostControl
          usageCents={deployUsage?.grossCents ?? null}
          isAdmin={isAdmin}
          showCompute={deployPlans?.configured !== false}
        />

        <RelatedPages workspaceSlug={workspace.slug} />
      </div>
    </Shell>
  );
};
