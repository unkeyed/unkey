"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { formatDollars, formatNumber } from "@/lib/fmt";
import { type DeployCheckoutOrigin, routes } from "@/lib/navigation/routes";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconClockRotateClockwiseOutline18,
  IconCodeBranchOutline18,
  IconEarthOutline18,
  IconEyeOutline18,
  IconLayers3Outline18,
  IconMicrochipOutline18,
  type IconProps,
  IconRamOutline18,
  IconUserOutline18,
} from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  Button,
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
  toast,
} from "@unkey/ui";
import { cn } from "cn";
import { type ComponentType, type ReactNode, useState } from "react";
import { currentApiProduct } from "./api-plan";
import {
  CREDITS_INFO,
  CREDITS_LINK_HREF,
  CREDITS_LINK_LABEL,
  PLAN_BLURBS,
} from "./compute-plan-copy";
import { ComputePlanConfirmDialog } from "./compute-plan-picker-v2";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { type PaywallProduct, type PaywallReason, paywallCopy } from "./paywall-copy";
import { PlanOptionList } from "./plan-change-modal";
import { type PlanFeatureKind, type PlanFeatureSet, computePlanFeatures } from "./plan-features";
import { PlanTierIcon } from "./plan-tier-icons";

type PlansScreenProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  reason: PaywallReason;
  from?: DeployCheckoutOrigin;
};

export function PlansScreen({ open, onOpenChange, reason, from = "billing" }: PlansScreenProps) {
  const { user } = useWorkspace();
  const isAdmin = user?.role === "admin";

  const { data: subscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });
  const currentPlan = subscription?.plan ?? null;

  const { data: plansData } = trpc.stripe.getDeployPlans.useQuery(undefined, {
    enabled: open,
    staleTime: 60_000,
    trpc: { context: { skipBatch: true } },
  });

  const copy = paywallCopy(reason);
  const products = copy.products.filter(
    (product) => product !== "compute" || plansData?.configured !== false,
  );

  const panels: Record<PaywallProduct, ReactNode> = {
    compute: (
      <ComputePlans
        plans={plansData?.plans}
        currentPlan={currentPlan}
        isAdmin={isAdmin}
        recommendedPlan={copy.recommendedPlan}
        from={from}
      />
    ),
    api: (
      <div className="mx-auto w-full max-w-[560px]">
        <ApiPlans isAdmin={isAdmin} enabled={open} />
      </div>
    ),
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="top-0 left-0 block h-dvh w-screen max-w-none translate-x-0 translate-y-0 overflow-y-auto rounded-none bg-background p-0 sm:rounded-none">
        <div className="mx-auto flex min-h-dvh w-full max-w-[1040px] flex-col items-center justify-center px-6 py-16">
          <Logo className="mb-6 h-6 w-auto" aria-hidden="true" />
          <DialogTitle className="text-center font-semibold text-2xl text-gray-12 tracking-[-0.03em]">
            {copy.title}
          </DialogTitle>
          <DialogDescription className="mt-2 max-w-lg text-balance text-center text-gray-11 text-sm leading-6">
            {copy.description}
          </DialogDescription>

          {products.length > 1 ? (
            <Tabs defaultValue={products[0]} className="mt-8 flex w-full flex-col items-center">
              <TabsList>
                {products.map((product) => (
                  <TabsTrigger key={product} value={product}>
                    {PRODUCT_LABELS[product]}
                  </TabsTrigger>
                ))}
              </TabsList>
              {products.map((product) => (
                <TabsContent key={product} value={product} className="mt-8 w-full">
                  {panels[product]}
                </TabsContent>
              ))}
            </Tabs>
          ) : (
            <div className="mt-8 w-full">{products[0] ? panels[products[0]] : null}</div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

const PRODUCT_LABELS: Record<PaywallProduct, string> = {
  compute: "Compute",
  api: "API",
};

const FEATURE_ICONS: Record<PlanFeatureKind, ComponentType<IconProps>> = {
  git: IconCodeBranchOutline18,
  preview: IconEyeOutline18,
  rollback: IconArrowDottedRotateAnticlockwiseOutline18,
  team: IconUserOutline18,
  cpu: IconMicrochipOutline18,
  memory: IconRamOutline18,
  domains: IconEarthOutline18,
  autoscale: IconLayers3Outline18,
  logs: IconClockRotateClockwiseOutline18,
};

function PlanFeatureList({
  featureSet,
  planName,
}: {
  featureSet: PlanFeatureSet;
  planName: (plan: DeployPlan) => string;
}) {
  return (
    <div className="flex flex-col gap-2.5">
      {featureSet.inheritsFrom ? (
        <span className="text-gray-11 text-sm">
          Everything in {planName(featureSet.inheritsFrom)}, plus
        </span>
      ) : null}
      <ul className="flex flex-col gap-2.5">
        {featureSet.features.map((feature) => {
          const Icon = FEATURE_ICONS[feature.kind];
          return (
            <li key={feature.label} className="flex items-center gap-2.5 text-gray-12 text-sm">
              <Icon className="size-4 shrink-0 text-gray-11" />
              {feature.label}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function intervalSuffix(interval: string | null): string {
  return interval === "year" ? "/yr" : "/mo";
}

function ComputePlans({
  plans,
  currentPlan,
  isAdmin,
  recommendedPlan,
  from,
}: {
  plans: DeployPlanOption[] | undefined;
  currentPlan: DeployPlan | null;
  isAdmin: boolean;
  recommendedPlan?: DeployPlan;
  from: DeployCheckoutOrigin;
}) {
  const workspace = useWorkspaceNavigation();
  const trpcUtils = trpc.useUtils();
  const [pendingPlan, setPendingPlan] = useState<DeployPlanOption | null>(null);
  const [startingCheckout, setStartingCheckout] = useState<DeployPlan | null>(null);

  const change = trpc.stripe.changeDeployPlan.useMutation({
    onSuccess: async (result) => {
      if (result.kind === "payment_required") {
        window.location.assign(result.paymentUrl);
        return;
      }
      setPendingPlan(null);
      toast.success("Compute plan changed");
      await Promise.all([
        trpcUtils.stripe.getDeploySubscription.invalidate(),
        trpcUtils.stripe.getDeployEntitlement.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      window.location.reload();
    },
    onError: (err) => toast.error(err.message),
  });

  if (!plans) {
    return (
      <div className="grid w-full gap-4 md:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-[420px] rounded-xl" />
        ))}
      </div>
    );
  }

  const currentAmount = plans.find((p) => p.plan === currentPlan)?.amount ?? null;

  const select = (option: DeployPlanOption) => {
    if (currentPlan) {
      setPendingPlan(option);
      return;
    }
    setStartingCheckout(option.plan);
    window.location.assign(
      routes.settings.stripe.checkout({
        workspaceSlug: workspace.slug,
        intent: "deploy",
        plan: option.plan,
        from,
      }),
    );
  };

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="grid w-full gap-4 md:grid-cols-3">
        {plans.map((option) => {
          const isCurrent = option.plan === currentPlan;
          const isRecommended = option.plan === recommendedPlan && !isCurrent;
          const isDowngrade =
            currentAmount !== null && option.amount !== null && option.amount < currentAmount;
          const label = isCurrent
            ? "Current plan"
            : currentPlan
              ? `${isDowngrade ? "Downgrade" : "Upgrade"} to ${option.name}`
              : `Choose ${option.name}`;

          return (
            <div
              key={option.plan}
              className={cn(
                "flex flex-col gap-5 rounded-xl border bg-raised p-6",
                isRecommended && "border-info-7 ring-1 ring-info-7",
              )}
            >
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between gap-2">
                  <span className="flex items-center gap-2 font-medium text-base text-gray-12">
                    <PlanTierIcon plan={option.plan} className="size-4" />
                    {option.name}
                  </span>
                  {isRecommended ? (
                    <span className="rounded-full bg-info-3 px-2 py-0.5 text-info-11 text-xs">
                      Recommended
                    </span>
                  ) : null}
                </div>
                <span className="text-gray-11 text-sm">{PLAN_BLURBS[option.plan]}</span>
              </div>

              <div className="flex items-baseline gap-1">
                {option.amount !== null ? (
                  <>
                    <span className="font-semibold text-3xl text-gray-12 tabular-nums">
                      {formatDollars(option.amount)}
                    </span>
                    <span className="text-gray-11 text-sm">{intervalSuffix(option.interval)}</span>
                  </>
                ) : (
                  <span className="font-semibold text-3xl text-gray-12">Contact us</span>
                )}
              </div>

              <Button
                variant={isRecommended ? "primary" : "outline"}
                size="lg"
                className="w-full"
                disabled={isCurrent || !isAdmin || startingCheckout !== null}
                loading={startingCheckout === option.plan}
                title={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
                onClick={() => select(option)}
              >
                {label}
              </Button>

              <PlanFeatureList
                featureSet={computePlanFeatures(option.plan)}
                planName={(plan) => plans.find((p) => p.plan === plan)?.name ?? plan}
              />
            </div>
          );
        })}
      </div>
      <AlertBanner className="w-auto self-center px-3 py-2">
        <AlertBannerDescription className="text-xs">
          {CREDITS_INFO}{" "}
          <a href={CREDITS_LINK_HREF} target="_blank" rel="noopener noreferrer">
            {CREDITS_LINK_LABEL}
          </a>
        </AlertBannerDescription>
      </AlertBanner>

      <ComputePlanConfirmDialog
        plan={pendingPlan}
        onOpenChange={(isOpen) => {
          if (!isOpen) {
            setPendingPlan(null);
          }
        }}
        onConfirm={() => {
          if (pendingPlan) {
            change.mutate({ plan: pendingPlan.plan });
          }
        }}
        isLoading={change.isLoading}
        currentPlanName={plans.find((p) => p.plan === currentPlan)?.name}
        note="Takes effect immediately. Upgrades are charged now and add the difference as usage credits; downgrades keep this period's credits, with the new fee starting next period."
      />
    </div>
  );
}

function ApiPlans({ isAdmin, enabled }: { isAdmin: boolean; enabled: boolean }) {
  const workspace = useWorkspaceNavigation();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data: billingInfo } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    enabled,
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  const onDone = (paymentUrl?: string | null) => {
    if (paymentUrl) {
      window.location.assign(paymentUrl);
      return;
    }
    toast.success("API plan activated");
    window.location.reload();
  };

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: (result) => {
      if (result.status === "checkout") {
        window.location.assign(result.checkoutUrl);
        return;
      }
      onDone(result.status === "payment_required" ? result.paymentUrl : null);
    },
    onError: (err) => toast.error(err.message),
  });
  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: (result) => onDone(result.kind === "payment_required" ? result.paymentUrl : null),
    onError: (err) => toast.error(err.message),
  });

  if (!billingInfo) {
    return <Skeleton className="h-[360px] w-full rounded-xl" />;
  }

  const currentProduct = currentApiProduct({
    products: billingInfo.products,
    subscription: billingInfo.subscription,
    currentProductId: billingInfo.currentProductId,
  });
  const currentId = currentProduct?.id ?? null;
  const selected = selectedId ?? currentId;
  const isSubmitting = createSubscription.isLoading || updateSubscription.isLoading;

  const subscribe = () => {
    if (!selected) {
      return;
    }
    if (!workspace.stripeCustomerId) {
      window.location.assign(
        routes.settings.stripe.checkout({ workspaceSlug: workspace.slug, intent: "api" }),
      );
      return;
    }
    if (currentProduct) {
      updateSubscription.mutate({ newProductId: selected });
      return;
    }
    createSubscription.mutate({ productId: selected });
  };

  return (
    <div className="flex flex-col gap-4">
      <p className="text-center text-gray-11 text-sm">
        Tiered plans for key verifications and ratelimits. Every API plan includes team members.
      </p>
      <PlanOptionList
        options={billingInfo.products.map((product) => ({
          id: product.id,
          name: product.name,
          amount: product.dollar * 100,
          interval: "month",
          detail: `${formatNumber(product.quotas.requestsPerMonth)} requests/month`,
        }))}
        currentId={currentId}
        selectedId={selected}
        onSelect={setSelectedId}
      />
      <Button
        variant="primary"
        size="xlg"
        className="w-full rounded-lg"
        disabled={!isAdmin || !selected || selected === currentId || isSubmitting}
        loading={isSubmitting}
        title={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
        onClick={subscribe}
      >
        {currentProduct ? "Change plan" : "Subscribe"}
      </Button>
    </div>
  );
}
