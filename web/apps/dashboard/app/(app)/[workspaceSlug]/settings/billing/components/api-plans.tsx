"use client";

import { announceUpgrade } from "@/components/billing/upgrade-success/upgrade-success-dialog";
import { currentApiProduct } from "@/lib/billing/api-plan";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { IconUserOutline18 } from "@unkey/icons";
import { Button, Skeleton, toast } from "@unkey/ui";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { CancelApiDialog, CancelPlanLink } from "./cancel-actions";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { PlanOptionList } from "./plan-option-list";

const API_PLAN_ROW_PLACEHOLDERS = ["a", "b", "c", "d", "e", "f", "g"];

const CHANGE_NOTE =
  "Upgrades take effect immediately and are prorated. Downgrades start next billing period; your current plan stays active and no refund is issued.";

type ApiPlansProps = {
  isAdmin: boolean;
  manage: boolean;
  usedThisMonth: number | null;
  onChanged: () => void;
};

export function ApiPlans({ isAdmin, manage, usedThisMonth, onChanged }: ApiPlansProps) {
  const router = useRouter();
  const pathname = usePathname();
  const trpcUtils = trpc.useUtils();
  const [isCancelOpen, setCancelOpen] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isRedirecting, setRedirecting] = useState(false);

  useEffect(() => {
    // Returning from Stripe via the back button restores this page from the
    // bfcache with the redirect spinner still set.
    const reset = (event: PageTransitionEvent) => {
      if (event.persisted) {
        setRedirecting(false);
      }
    };
    window.addEventListener("pageshow", reset);
    return () => window.removeEventListener("pageshow", reset);
  }, []);

  const leaveFor = (url: string) => {
    setRedirecting(true);
    window.location.assign(url);
  };

  const { data: billingInfo, isError } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: (result) => {
      leaveFor(result.status === "checkout" ? result.checkoutUrl : result.paymentUrl);
    },
    onError: (err) => toast.error(err.message),
  });
  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: async (result) => {
      if (result.kind === "payment_required") {
        leaveFor(result.paymentUrl);
        return;
      }
      if (result.kind === "scheduled") {
        toast.success(
          `API plan downgrade scheduled for ${new Date(result.effectiveAt).toLocaleDateString()}`,
        );
      }
      await Promise.all([
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
        trpcUtils.billing.queryUsage.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
      router.refresh();
      onChanged();
      if (result.kind === "applied") {
        announceUpgrade({ kind: "api" });
      }
    },
    onError: (err) => toast.error(err.message),
  });

  const currentProduct = billingInfo
    ? currentApiProduct({
        products: billingInfo.products,
        subscription: billingInfo.subscription,
        currentProductId: billingInfo.currentProductId,
      })
    : undefined;
  const currentId = currentProduct?.id ?? null;
  const selected = selectedId ?? currentId;
  const isSubmitting =
    createSubscription.isLoading || updateSubscription.isLoading || isRedirecting;
  const selectedProduct = billingInfo?.products.find((product) => product.id === selected);
  const warning =
    selectedProduct &&
    selected !== currentId &&
    usedThisMonth !== null &&
    usedThisMonth > selectedProduct.quotas.requestsPerMonth
      ? `Your usage this month (${formatNumber(usedThisMonth)}) already exceeds the ${formatNumber(selectedProduct.quotas.requestsPerMonth)} requests ${selectedProduct.name} includes.`
      : null;
  const canCancel = Boolean(
    billingInfo?.subscription &&
      billingInfo.subscription.status === "active" &&
      !billingInfo.subscription.cancelAt,
  );

  if (isError) {
    return (
      <p className="text-center text-gray-11 text-sm">
        API plans could not be loaded. Reload the page or contact support@unkey.com.
      </p>
    );
  }

  const subscribe = () => {
    if (!selected) {
      return;
    }
    if (currentProduct) {
      updateSubscription.mutate({ newProductId: selected });
      return;
    }
    createSubscription.mutate({ productId: selected, returnTo: pathname });
  };

  return (
    <div className="flex flex-col gap-4">
      {billingInfo ? (
        <PlanOptionList
          options={billingInfo.products.map((product) => ({
            id: product.id,
            name: `${formatNumber(product.quotas.requestsPerMonth)} requests`,
            monthlyCents: product.dollar * 100,
          }))}
          currentId={currentId}
          selectedId={selected}
          onSelect={setSelectedId}
          label="API plans"
        />
      ) : (
        <div className="flex flex-col divide-y overflow-hidden rounded-xl border bg-raised">
          {API_PLAN_ROW_PLACEHOLDERS.map((key) => (
            <div key={key} className="flex h-[45px] items-center px-4">
              <Skeleton className="h-4 w-full rounded" />
            </div>
          ))}
        </div>
      )}
      <Button
        variant="primary"
        size="xlg"
        className="w-full rounded-lg"
        disabled={!billingInfo || !isAdmin || !selected || selected === currentId || isSubmitting}
        loading={isSubmitting}
        title={isAdmin ? undefined : ADMIN_ONLY_TOOLTIP}
        onClick={subscribe}
      >
        {currentProduct ? "Change plan" : "Subscribe"}
      </Button>
      <output className="block text-sm text-warning-11 leading-5 empty:hidden">{warning}</output>
      {currentProduct ? (
        <p className="text-center text-gray-11 text-xs leading-5">{CHANGE_NOTE}</p>
      ) : null}
      {manage && canCancel ? (
        <div className="flex justify-center">
          <CancelPlanLink isAdmin={isAdmin} onClick={() => setCancelOpen(true)}>
            Cancel API plan
          </CancelPlanLink>
        </div>
      ) : null}
      <CancelApiDialog open={isCancelOpen} onOpenChange={setCancelOpen} />
      <p className="flex items-center justify-center gap-2 text-gray-11 text-sm">
        <IconUserOutline18 className="size-4" />
        All API plans include unlimited team members.
      </p>
    </div>
  );
}
