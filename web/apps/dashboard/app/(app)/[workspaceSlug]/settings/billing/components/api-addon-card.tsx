"use client";

import { formatNumber } from "@/lib/fmt";
import { formatMs } from "@/lib/ms";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { Nodes, TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, InfoTooltip, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { PlanChangeModal } from "./plan-change-modal";
import { ProductCard } from "./product-card";
import { UsageMeter } from "./usage-meter";

const NEEDS_PAYMENT_TOOLTIP = "Add a payment method before upgrading the API plan";
const FREE_TIER_QUOTA = 150_000;

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type ApiAddOnCardProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  /** Open the plan picker on mount (post-checkout intent hand-off). */
  autoOpenPlanModal?: boolean;
};

/**
 * API management as an add-on: key verification and ratelimit plan with its
 * quota meter. Sits below the Deploy hero; same card language, its own accent.
 * Cancelling is a quiet footer link with the existing downgrade warning.
 */
export const ApiAddOnCard: React.FC<ApiAddOnCardProps> = ({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  products,
  subscription,
  currentProductId,
  autoOpenPlanModal = false,
}) => {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();
  const [showPlanModal, setShowPlanModal] = useState(autoOpenPlanModal);
  const [isCancelOpen, setCancelOpen] = useState(false);

  const { data: usage } = trpc.billing.queryUsage.useQuery(undefined, {
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
    retry: 1,
  });

  const revalidate = async () => {
    await Promise.all([
      trpcUtils.workspace.getCurrent.invalidate(),
      trpcUtils.billing.queryUsage.invalidate(),
      trpcUtils.stripe.getBillingInfo.invalidate(),
      trpcUtils.stripe.getUpcomingInvoice.invalidate(),
    ]);
  };

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: async () => {
      setShowPlanModal(false);
      toast.success("Plan activated");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });
  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: async () => {
      setShowPlanModal(false);
      toast.success("API plan changed");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const uncancelSubscription = trpc.stripe.uncancelSubscription.useMutation({
    onSuccess: async () => {
      await revalidate();
      router.refresh();
      toast.info("API plan resumed");
    },
    onError: () => {
      toast.error("Failed to resume the plan. Please try again or contact support@unkey.com.");
    },
  });

  const cancelSubscription = trpc.stripe.cancelSubscription.useMutation({
    onSuccess: async () => {
      await revalidate();
      router.refresh();
      setCancelOpen(false);
      toast.info("Subscription cancelled");
    },
    onError: (err) => toast.error(err.message),
  });

  const hasPaidSubscription = Boolean(
    subscription &&
      currentProductId &&
      ["active", "trialing", "past_due"].includes(subscription.status),
  );
  const currentProduct = hasPaidSubscription
    ? products.find((p) => p.id === currentProductId)
    : undefined;
  const allowCancel = subscription && subscription.status === "active" && !subscription.cancelAt;
  // A scheduled cancellation can only ever affect the API plan: Compute is
  // cancelled immediately, never at period end, and cancelSubscription
  // refuses to schedule while Compute items exist. So the notice lives here
  // inside the API card, not at page level.
  const cancelAt =
    subscription?.cancelAt && subscription.cancelAt > Date.now()
      ? subscription.cancelAt
      : undefined;

  const quota = currentProduct?.quotas.requestsPerMonth ?? FREE_TIER_QUOTA;
  const used = (usage?.billableVerifications ?? 0) + (usage?.billableRatelimits ?? 0);

  const upgradeDisabled = !isAdmin || !hasPaymentMethod;
  const upgradeTooltip = isAdmin ? NEEDS_PAYMENT_TOOLTIP : ADMIN_ONLY_TOOLTIP;

  return (
    <>
      <ProductCard
        icon={<Nodes iconSize="md-regular" />}
        iconClassName="bg-infoA-3 text-info-11"
        name="API Management"
        tag={currentProduct ? currentProduct.name : "Free"}
        subtitle={
          currentProduct
            ? `$${currentProduct.dollar}/month, ${formatNumber(quota)} requests included`
            : `Key verifications and ratelimits, ${formatNumber(FREE_TIER_QUOTA)} requests free per month`
        }
        action={
          currentProduct ? (
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <Button
                  variant="outline"
                  size="md"
                  disabled={!isAdmin}
                  onClick={() => setShowPlanModal(true)}
                >
                  Change plan
                </Button>
              </span>
            </InfoTooltip>
          ) : (
            <InfoTooltip content={upgradeTooltip} disabled={!upgradeDisabled} asChild>
              <span>
                <Button
                  variant="outline"
                  size="md"
                  disabled={upgradeDisabled}
                  onClick={() => {
                    if (hasPaymentMethod) {
                      setShowPlanModal(true);
                    } else {
                      router.push(
                        routes.settings.stripe.checkout({ workspaceSlug, intent: "api" }),
                      );
                    }
                  }}
                >
                  Upgrade
                </Button>
              </span>
            </InfoTooltip>
          )
        }
        footer={
          allowCancel ? (
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <button
                  type="button"
                  className="text-[13px] text-gray-9 transition-colors hover:text-gray-11 disabled:cursor-not-allowed"
                  disabled={!isAdmin}
                  onClick={() => setCancelOpen(true)}
                >
                  Cancel plan
                </button>
              </span>
            </InfoTooltip>
          ) : undefined
        }
      >
        <div className="flex flex-col gap-4">
          {cancelAt ? (
            <div className="flex items-center justify-between gap-4 rounded-lg border border-warningA-6 bg-warningA-2 px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <TriangleWarning2 iconSize="md-regular" className="shrink-0 text-warning-11" />
                <p className="truncate text-[13px] text-gray-11">
                  Your API plan ends in {formatMs(cancelAt - Date.now(), { long: true })} on{" "}
                  {new Date(cancelAt).toLocaleDateString()}; the workspace then downgrades to the
                  free tier.
                </p>
              </div>
              <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                <span>
                  <Button
                    variant="outline"
                    size="md"
                    loading={uncancelSubscription.isLoading}
                    disabled={!isAdmin || uncancelSubscription.isLoading}
                    onClick={() => uncancelSubscription.mutate()}
                  >
                    Resubscribe
                  </Button>
                </span>
              </InfoTooltip>
            </div>
          ) : null}
          <UsageMeter
            label="Verifications & ratelimits this month"
            value={usage ? `${formatNumber(used)} / ${formatNumber(quota)}` : "—"}
            fraction={usage && quota > 0 ? used / quota : null}
            fillClassName="bg-info-9"
          />
        </div>
      </ProductCard>

      {hasPaymentMethod ? (
        <PlanChangeModal
          isOpen={showPlanModal}
          onOpenChange={setShowPlanModal}
          title={currentProduct ? "Change API plan" : "Choose an API plan"}
          subTitle="Tiered plans for key verifications and ratelimits."
          options={products.map((product) => ({
            id: product.id,
            name: product.name,
            // Catalog products are priced in whole dollars per month.
            amount: product.dollar * 100,
            interval: "month",
            // Compact count for the inline row: "1M requests/month".
            detail: `${formatNumber(product.quotas.requestsPerMonth)} requests/month`,
          }))}
          currentId={currentProduct?.id ?? null}
          // Informational: downgrading below this month's usage means requests
          // beyond the smaller quota get rejected once it is exhausted.
          warningFor={(option) => {
            const target = products.find((p) => p.id === option.id);
            return target && used > target.quotas.requestsPerMonth
              ? `Your usage this month (${formatNumber(used)}) already exceeds the ${formatNumber(
                  target.quotas.requestsPerMonth,
                )} requests ${target.name} includes.`
              : null;
          }}
          changeNote="Takes effect immediately; the prorated difference is invoiced right away."
          submittingId={
            createSubscription.isLoading
              ? createSubscription.variables?.productId
              : updateSubscription.isLoading
                ? updateSubscription.variables?.newProductId
                : undefined
          }
          onSelect={(id) => {
            if (currentProduct) {
              updateSubscription.mutate({ newProductId: id });
            } else {
              createSubscription.mutate({ productId: id });
            }
          }}
        />
      ) : null}

      <DialogContainer
        isOpen={isCancelOpen}
        onOpenChange={setCancelOpen}
        title="Cancel API plan"
        subTitle="Downgrade your workspace to the free tier"
        footer={
          <div className="flex w-full flex-col items-center justify-center gap-2">
            <Button
              type="button"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              loading={cancelSubscription.isLoading}
              onClick={() => cancelSubscription.mutate()}
            >
              Cancel API plan
            </Button>
            <div className="text-gray-9 text-xs">
              You can resume your subscription until the end of the billing period
            </div>
          </div>
        }
      >
        <div className="flex items-center gap-4 rounded-xl border border-errorA-3 bg-errorA-2 px-[22px] py-6 dark:bg-black">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-error-9">
            <TriangleWarning2 iconSize="sm-regular" className="text-white" />
          </div>
          <div className="text-[13px] text-error-12 leading-6">
            <span className="font-medium">Warning:</span> cancelling your API plan will downgrade
            your workspace to the free tier at the end of the current billing period. You will lose
            access to paid features, usage limits will be reduced, and all team members other than
            you will be deactivated.
          </div>
        </div>
      </DialogContainer>
    </>
  );
};
