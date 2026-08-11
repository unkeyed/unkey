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
  PageHeaderTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { BillingNotices } from "./components/billing-notices";
import { CostControl } from "./components/cost-control";
import { InvoiceCard } from "./components/invoice-card";
import { Phone } from "./components/phone-icon";
import { PlansCard } from "./components/plans-card";
import { RelatedPages } from "./components/related-pages";

const SALES_CALL = "https://cal.com/james-r-perkins/sales";
const SUPPORT_MAILTO = "mailto:support@unkey.com";

function Shell({ children }: { children: ReactNode }) {
  return (
    <PageContainer>
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Billing</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>
          <Button
            variant="outline"
            size="md"
            render={<Link href={SALES_CALL} target="_blank" rel="noopener noreferrer" />}
          >
            <Phone iconSize="md-medium" />
            Schedule a call
          </Button>
          <Button variant="outline" size="md" render={<Link href={SUPPORT_MAILTO} />}>
            Contact us
          </Button>
        </PageHeaderActions>
      </PageHeader>
      <PageBody>{children}</PageBody>
    </PageContainer>
  );
}

function isCheckoutIntent(value: string | null): value is "compute" | "api" {
  return value === "compute" || value === "api";
}

/**
 * Billing page shown when the deployBilling flag is on: the upcoming invoice,
 * then the plan per product, then cost control split per product, then the pages
 * that answer the questions this one deliberately does not — Usage for
 * consumption, the docs for how any of it works. Cost control lives here rather
 * than on Limits because a budget is a billing preference, not a ceiling we
 * impose. Flag-off keeps the existing single-product page (./client).
 */
export function DeployBillingClient() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const searchParams = useSearchParams();

  // Post-checkout hand-off: /success sends the user back here with the intent
  // their checkout round-trip started with, and we reopen the plan picker they
  // were heading for. Captured once and stripped from the URL so refreshes don't
  // reopen modals.
  const [checkoutIntent] = useState(() => {
    const intent = searchParams?.get("intent") ?? null;
    return isCheckoutIntent(intent) ? intent : null;
  });
  // Guarded with a ref because router.replace is async: searchParams stays
  // populated until the navigation commits, so a re-render in that window would
  // otherwise re-fire the replace.
  const intentCleared = useRef(false);
  useEffect(() => {
    if (checkoutIntent && !intentCleared.current) {
      intentCleared.current = true;
      router.replace(routes.settings.billing({ workspaceSlug: workspace.slug }));
    }
  }, [checkoutIntent, router, workspace.slug]);

  // Server-side `requireWorkspaceAdmin` enforces this on every billing
  // mutation; we mirror it on the client purely for UX so non-admin members get
  // a clear "admin required" affordance instead of a request that fails with
  // FORBIDDEN.
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

  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });
  const { data: deployPlans } = trpc.stripe.getDeployPlans.useQuery(undefined, {
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });

  const subscription = billingInfo?.subscription;
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);
  const currentApiProduct = billingInfo?.products.find(
    (p) => p.id === billingInfo.currentProductId,
  );

  return (
    <Shell>
      <BillingNotices isAdmin={isAdmin} subscription={subscription} />

      <InvoiceCard
        workspaceSlug={workspace.slug}
        isAdmin={isAdmin}
        hasPaymentMethod={hasPaymentMethod}
      />

      {billingError ? (
        <Empty>
          <Empty.Title>Failed to load API billing information</Empty.Title>
          <Empty.Description>
            There was an error loading your API billing information. Please try again later.
          </Empty.Description>
        </Empty>
      ) : billingLoading || !billingInfo ? (
        <div className="h-[140px] w-full animate-pulse rounded-lg bg-grayA-3" />
      ) : (
        <PlansCard
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
          workspaceSlug={workspace.slug}
          products={billingInfo.products}
          subscription={subscription}
          currentProductId={billingInfo.currentProductId}
          checkoutIntent={checkoutIntent}
        />
      )}

      {/* Nothing to budget without a Compute subscription, so the group only
          exists once there is metered spend to manage. */}
      {deploySubscription?.plan ? (
        <CostControl
          isAdmin={isAdmin}
          showCompute={deployPlans?.configured !== false}
          apiFeeCents={(currentApiProduct?.dollar ?? 0) * 100}
        />
      ) : null}

      <RelatedPages workspaceSlug={workspace.slug} />
    </Shell>
  );
}
