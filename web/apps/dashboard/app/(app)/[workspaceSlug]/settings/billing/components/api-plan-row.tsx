"use client";

import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { Nodes } from "@unkey/icons";
import { Item, ItemMedia, ItemTitle, toast } from "@unkey/ui";
import { useState } from "react";
import { currentApiProduct } from "./api-plan";
import { CancelApiDialog, CancelPlanLink } from "./cancel-actions";
import { PlanChangeModal } from "./plan-change-modal";
import { PLAN_COLUMNS, PlanName, PlanPrice, PlanRowAction } from "./plan-row";

const NEEDS_PAYMENT_TOOLTIP = "Add a payment method before upgrading the API plan";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type ApiPlanRowProps = {
  isAdmin: boolean | undefined;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  emphasize: boolean;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  autoOpenPlanModal?: boolean;
};

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
      toast.success(
        result.kind === "scheduled"
          ? `API plan downgrade scheduled for ${new Date(result.effectiveAt).toLocaleDateString()}`
          : "API plan changed",
      );
      await revalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const currentProduct = currentApiProduct({ products, subscription, currentProductId });

  const used = (usage?.billableVerifications ?? 0) + (usage?.billableRatelimits ?? 0);

  const canCancel = Boolean(
    subscription && subscription.status === "active" && !subscription.cancelAt,
  );

  return (
    <>
      <Item className={PLAN_COLUMNS}>
        <div className="flex min-w-0 items-center gap-3">
          <ItemMedia className="bg-infoA-3 text-info-11">
            <Nodes />
          </ItemMedia>
          <ItemTitle className="truncate">API management</ItemTitle>
        </div>
        <PlanName>{currentProduct ? currentProduct.name : "Free"}</PlanName>
        <PlanPrice feeCents={(currentProduct?.dollar ?? 0) * 100} />
        <span className="flex justify-end">
          <PlanRowAction
            isAdmin={isAdmin}
            hasPlan={currentProduct !== undefined}
            hasPaymentMethod={hasPaymentMethod}
            needsPaymentReason={NEEDS_PAYMENT_TOOLTIP}
            emphasize={emphasize}
            onClick={() => setShowPlanModal(true)}
            chooseLabel="Upgrade"
          />
        </span>
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
            amount: product.dollar * 100,
            interval: "month",
            detail: `${formatNumber(product.quotas.requestsPerMonth)} requests/month`,
          }))}
          currentId={currentProduct?.id ?? null}
          warningFor={(option) => {
            const target = products.find((p) => p.id === option.id);
            return target && used > target.quotas.requestsPerMonth
              ? `Your usage this month (${formatNumber(used)}) already exceeds the ${formatNumber(
                  target.quotas.requestsPerMonth,
                )} requests ${target.name} includes.`
              : null;
          }}
          changeNote="Upgrades take effect immediately and are prorated. Downgrades start next billing period; your current plan stays active and no refund is issued."
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
          cancelAction={
            canCancel ? (
              <CancelPlanLink
                isAdmin={isAdmin}
                onClick={() => {
                  setShowPlanModal(false);
                  setCancelOpen(true);
                }}
              >
                Cancel API plan
              </CancelPlanLink>
            ) : null
          }
        />
      ) : null}

      <CancelApiDialog open={isCancelOpen} onOpenChange={setCancelOpen} />
    </>
  );
}
