"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { currentPlanState } from "@/lib/billing/plan-card-state";
import { useFlag } from "@/lib/flags/provider";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
  Logo,
  Skeleton,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";
import Link from "next/link";
import { type ReactNode, useState } from "react";
import { ApiPlans } from "./api-plans";
import { CREDITS_INFO, CREDITS_LINK_HREF, CREDITS_LINK_LABEL } from "./compute-plan-copy";
import { ComputePlans } from "./compute-plans";
import {
  type PaywallProduct,
  type PaywallReason,
  availableProducts,
  defaultProduct,
  paywallCopy,
} from "./paywall-copy";

type PlansScreenProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  reason: PaywallReason;
  from?: DeployCheckoutOrigin;
};

export function PlansScreen({ open, onOpenChange, reason, from = "billing" }: PlansScreenProps) {
  const { user } = useWorkspace();
  const isAdmin = user?.role === "admin";
  const deployBilling = useFlag("deployBilling");
  const workspace = useWorkspaceNavigation();

  const copy = paywallCopy(reason);
  const wantsCompute = open && deployBilling && copy.products.includes("compute");
  const wantsApi = open && copy.products.includes("api");

  const subscriptionQuery = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    enabled: wantsCompute,
    staleTime: 30_000,
  });
  const current = currentPlanState({
    isError: subscriptionQuery.isError,
    isLoading: subscriptionQuery.isLoading,
    plan: subscriptionQuery.data?.plan ?? null,
  });

  const plansQuery = trpc.stripe.getDeployPlans.useQuery(undefined, {
    enabled: wantsCompute,
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });
  const deployUsageQuery = trpc.billing.queryDeployUsage.useQuery(undefined, {
    enabled: wantsCompute,
    staleTime: 60_000,
    retry: 1,
    trpc: { context: { skipBatch: true } },
  });
  const apiUsageQuery = trpc.billing.queryUsage.useQuery(undefined, {
    enabled: wantsApi,
    staleTime: 60_000,
    retry: 1,
    trpc: { context: { skipBatch: true } },
  });

  const products = availableProducts(copy.products, {
    computeEnabled: deployBilling && !plansQuery.isError && plansQuery.data?.configured !== false,
  });

  const [selectedProduct, setSelectedProduct] = useState<PaywallProduct | null>(null);
  const [initialProduct, setInitialProduct] = useState<PaywallProduct | null>(null);
  const choiceSettled =
    open &&
    (!wantsCompute || (!plansQuery.isLoading && !deployUsageQuery.isLoading)) &&
    (!wantsApi || !apiUsageQuery.isLoading);
  if (initialProduct === null && products.length > 0 && choiceSettled) {
    setInitialProduct(
      defaultProduct(products, {
        compute: deployUsageQuery.data?.grossCents ?? 0,
        api: apiUsageQuery.data?.billableTotal ?? 0,
      }) ?? null,
    );
  }
  const chosenProduct = [selectedProduct, initialProduct].find(
    (product): product is PaywallProduct => product !== null && products.includes(product),
  );
  const activeProduct = products.length > 1 ? (chosenProduct ?? null) : (products[0] ?? null);

  const close = () => onOpenChange(false);
  const panels: Record<PaywallProduct, ReactNode> = {
    compute: (
      <ComputePlans
        plans={copy.computePlans}
        options={plansQuery.data?.plans}
        current={current}
        usageCents={deployUsageQuery.data?.grossCents ?? null}
        isAdmin={isAdmin}
        from={from}
        manage={copy.manage}
        onChanged={close}
      />
    ),
    api: (
      <div className="mx-auto w-full max-w-[560px]">
        <ApiPlans
          isAdmin={isAdmin}
          manage={copy.manage}
          usedThisMonth={apiUsageQuery.data?.billableTotal ?? null}
          onChanged={close}
        />
      </div>
    ),
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="group/plans top-0 left-0 block h-dvh w-screen max-w-none translate-x-0 translate-y-0 overflow-hidden rounded-none bg-background p-0 transition-opacity duration-200 ease-[cubic-bezier(0.23,1,0.32,1)] data-ending-style:scale-100 data-starting-style:scale-100 data-ending-style:duration-150 motion-reduce:transition-none sm:rounded-none [&>button[aria-label='Close_dialog']]:top-8 [&>button[aria-label='Close_dialog']]:right-8">
        <div className="h-full overflow-y-auto">
          <div className="overflow-clip">
            <Tabs
              value={activeProduct}
              onValueChange={(value) =>
                setSelectedProduct(products.find((product) => product === value) ?? null)
              }
              className="mx-auto flex min-h-dvh w-full max-w-[1040px] flex-col items-center justify-start px-4 pt-24 pb-10 transition-[translate] duration-300 ease-[cubic-bezier(0.23,1,0.32,1)] group-data-starting-style/plans:translate-y-3 motion-reduce:transition-none md:px-6 md:pt-28 md:pb-16"
            >
              <div className="relative">
                <Logo
                  aria-hidden="true"
                  className="absolute bottom-full left-1/2 mb-10 h-6 w-auto -translate-x-1/2"
                />
                <DialogTitle className="text-center font-semibold text-gray-12 text-xl tracking-[-0.03em] md:text-2xl">
                  {copy.title}
                </DialogTitle>
              </div>
              <DialogDescription className="mt-2 max-w-md text-balance text-center text-gray-11 text-sm leading-6">
                {copy.description}
                {products.includes("compute") ? (
                  <span
                    aria-hidden={activeProduct !== "compute"}
                    className={activeProduct === "compute" ? undefined : "invisible"}
                  >
                    {" "}
                    {CREDITS_INFO}{" "}
                    <a
                      href={CREDITS_LINK_HREF}
                      target="_blank"
                      rel="noopener noreferrer"
                      tabIndex={activeProduct === "compute" ? undefined : -1}
                      className="underline underline-offset-2 hover:text-gray-12"
                    >
                      {CREDITS_LINK_LABEL}
                    </a>
                  </span>
                ) : null}
              </DialogDescription>
              {isAdmin ? null : (
                <p className="mt-3 rounded-lg bg-grayA-3 px-3 py-1.5 text-center text-gray-12 text-sm">
                  Only workspace admins can change plans. Ask an admin to upgrade.
                </p>
              )}
              {products.length > 1 ? (
                <TabsList className="mt-8 w-64">
                  {products.map((product) => (
                    <TabsTrigger key={product} value={product} className="flex-1">
                      {PRODUCT_LABELS[product]}
                    </TabsTrigger>
                  ))}
                </TabsList>
              ) : null}
              {products.map((product) => (
                <TabsContent key={product} value={product} className="mt-8 w-full">
                  {panels[product]}
                </TabsContent>
              ))}
              {activeProduct === null && products.length > 1 ? (
                <Skeleton className="mt-8 h-96 w-full max-w-[560px] rounded-xl" />
              ) : null}
              {products.length === 0 ? (
                <p className="mt-8 text-center text-gray-11 text-sm">
                  Plans aren't available right now.{" "}
                  <Link
                    href={routes.settings.billing({ workspaceSlug: workspace.slug })}
                    className="underline underline-offset-2 hover:text-gray-12"
                  >
                    Go to billing
                  </Link>
                </p>
              ) : null}
            </Tabs>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

const PRODUCT_LABELS: Record<PaywallProduct, string> = {
  compute: "Compute",
  api: "API",
};
