"use client";

import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { Nodes } from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ADMIN_ONLY_TOOLTIP, FREE_TIER_QUOTA } from "./constants";
import { PlanChangeModal } from "./plan-change-modal";

const NEEDS_PAYMENT_TOOLTIP = "Add a payment method before upgrading the API plan";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type ApiPlanRowProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  /** Primary styling, used only while nothing at all is subscribed. */
  emphasize: boolean;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  /** Open the plan picker on mount (post-checkout intent hand-off). */
  autoOpenPlanModal?: boolean;
};

/**
 * API management in the Plans card. A flat monthly fee for an included request
 * quota, so the row states the quota and the fee and nothing else: how much of
 * the quota is consumed is a Usage question, and the ceiling is on Limits.
 */
export function ApiPlanRow({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  emphasize,
  products,
  subscription,
  currentProductId,
  autoOpenPlanModal = false,
}: ApiPlanRowProps) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();
  const [showPlanModal, setShowPlanModal] = useState(autoOpenPlanModal);

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
    onSuccess: async (result) => {
      if (result.status === "checkout") {
        window.location.assign(result.checkoutUrl);
        return;
      }
      if (result.status === "payment_required") {
        window.location.assign(
          result.paymentUrl ?? routes.settings.stripe.checkout({ workspaceSlug, intent: "api" }),
        );
        return;
      }
      setShowPlanModal(false);
      toast.success("Plan activated");
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: async (result) => {
      if (result.kind === "payment_required") {
        window.location.assign(result.paymentUrl);
        return;
      }
      setShowPlanModal(false);
      toast.success("API plan changed");
      await revalidate();
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

  const quota = currentProduct?.quotas.requestsPerMonth ?? FREE_TIER_QUOTA;
  const used = (usage?.billableVerifications ?? 0) + (usage?.billableRatelimits ?? 0);

  const upgradeDisabled = !isAdmin || !hasPaymentMethod;

  return (
    <>
      <Item>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <Nodes />
        </ItemMedia>
        <ItemContent>
          <div className="flex h-4 items-center">
            <ItemTitle>API management</ItemTitle>
          </div>
          <ItemDescription>{formatNumber(quota)} requests included</ItemDescription>
        </ItemContent>
        <ItemActions className="gap-4">
          <span className="w-48 text-right tabular-nums">
            <span className="text-gray-11">{currentProduct ? currentProduct.name : "Free"} - </span>
            <span className="font-medium text-gray-12">
              ${currentProduct ? currentProduct.dollar : 0}/month
            </span>
          </span>
          <span className="flex w-32 justify-end">
            {currentProduct ? (
              <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
                <span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!isAdmin}
                    onClick={() => setShowPlanModal(true)}
                  >
                    Change
                  </Button>
                </span>
              </InfoTooltip>
            ) : (
              <InfoTooltip
                content={isAdmin ? NEEDS_PAYMENT_TOOLTIP : ADMIN_ONLY_TOOLTIP}
                disabled={!upgradeDisabled}
                asChild
              >
                <span>
                  <Button
                    variant={emphasize ? "primary" : "outline"}
                    size="sm"
                    disabled={upgradeDisabled}
                    onClick={() => {
                      if (hasPaymentMethod) {
                        setShowPlanModal(true);
                        return;
                      }
                      router.push(
                        routes.settings.stripe.checkout({ workspaceSlug, intent: "api" }),
                      );
                    }}
                  >
                    Upgrade
                  </Button>
                </span>
              </InfoTooltip>
            )}
          </span>
        </ItemActions>
      </Item>

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
              return;
            }
            createSubscription.mutate({ productId: id });
          }}
        />
      ) : null}
    </>
  );
}
