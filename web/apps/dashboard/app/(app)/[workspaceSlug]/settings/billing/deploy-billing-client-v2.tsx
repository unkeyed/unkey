"use client";

import { useConsumedSearchParam } from "@/hooks/use-consumed-search-param";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { SUPPORT_MAILTO } from "@/lib/support";
import { trpc } from "@/lib/trpc/client";
import { Phone } from "@unkey/icons";
import {
  Button,
  Empty,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { currentApiProduct } from "./components/api-plan";
import { BillingNotices } from "./components/billing-notices";
import { CostControl } from "./components/cost-control";
import { InvoiceCard } from "./components/invoice-card";
import { PlansCard } from "./components/plans-card";
import { RelatedPages } from "./components/related-pages";

const SALES_CALL_URL = "https://cal.com/james-r-perkins/sales";

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
            render={<Link href={SALES_CALL_URL} target="_blank" rel="noopener noreferrer" />}
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

export function DeployBillingClientV2() {
  const workspace = useWorkspaceNavigation();

  const checkoutIntent = useConsumedSearchParam(
    "intent",
    (value) => (isCheckoutIntent(value) ? value : null),
    routes.settings.billing({ workspaceSlug: workspace.slug }),
  );

  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
  const isAdmin = currentUser ? currentUser.role === "admin" : undefined;

  const { data: billingInfo, error: billingError } = trpc.stripe.getBillingInfo.useQuery(
    undefined,
    {
      staleTime: 30_000,
      trpc: { context: { skipBatch: true } },
    },
  );

  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });

  const subscription = billingInfo?.subscription;
  const hasPaymentMethod = Boolean(workspace.stripeCustomerId);
  const apiFeeCents = billingInfo
    ? (currentApiProduct({
        products: billingInfo.products,
        subscription,
        currentProductId: billingInfo.currentProductId,
      })?.dollar ?? 0) * 100
    : null;

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
      ) : billingInfo ? (
        <PlansCard
          isAdmin={isAdmin}
          hasPaymentMethod={hasPaymentMethod}
          workspaceSlug={workspace.slug}
          products={billingInfo.products}
          subscription={subscription}
          currentProductId={billingInfo.currentProductId}
          checkoutIntent={checkoutIntent}
        />
      ) : (
        <Skeleton className="h-[140px] w-full rounded-lg" />
      )}

      {deploySubscription?.plan ? (
        <CostControl isAdmin={isAdmin} apiFeeCents={apiFeeCents} />
      ) : null}

      <RelatedPages workspaceSlug={workspace.slug} />
    </Shell>
  );
}
