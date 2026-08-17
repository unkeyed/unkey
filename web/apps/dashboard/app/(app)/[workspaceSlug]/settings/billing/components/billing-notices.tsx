"use client";

import { formatDate } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { TriangleWarning2 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  toast,
} from "@unkey/ui";
import { useRouter } from "next/navigation";
import { AdminGate } from "./admin-gate";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

const PAYMENT_REQUIRED_STATUSES = ["incomplete", "incomplete_expired", "unpaid", "past_due"];

export function BillingNotices({
  isAdmin,
  subscription,
}: {
  isAdmin: boolean | undefined;
  subscription?: BillingInfo["subscription"];
}) {
  return (
    <>
      {subscription && PAYMENT_REQUIRED_STATUSES.includes(subscription.status) ? (
        <PaymentRequired status={subscription.status} />
      ) : null}
      {subscription?.cancelAt && subscription.cancelAt > Date.now() ? (
        <ScheduledCancellation isAdmin={isAdmin} cancelAt={subscription.cancelAt} />
      ) : null}
    </>
  );
}

function PaymentRequired({ status }: { status: string }) {
  const completePayment = trpc.stripe.getSubscriptionPaymentUrl.useMutation({
    onSuccess: ({ paymentUrl }) => window.location.assign(paymentUrl),
    onError: (err) => toast.error(err.message),
  });

  return (
    <AlertBanner variant="error">
      <TriangleWarning2 iconSize="md-regular" />
      <AlertBannerTitle>Payment required</AlertBannerTitle>
      <AlertBannerDescription>
        {status === "incomplete_expired"
          ? "The previous payment attempt expired. Choose your plan again to retry."
          : "Complete the payment in Stripe to activate or restore your plan."}
      </AlertBannerDescription>
      {status === "incomplete_expired" ? null : (
        <AlertBannerActions>
          <Button
            variant="outline"
            size="md"
            loading={completePayment.isLoading}
            onClick={() => completePayment.mutate()}
          >
            Complete payment
          </Button>
        </AlertBannerActions>
      )}
    </AlertBanner>
  );
}

function ScheduledCancellation({
  isAdmin,
  cancelAt,
}: {
  isAdmin: boolean | undefined;
  cancelAt: number;
}) {
  const router = useRouter();
  const trpcUtils = trpc.useUtils();

  const uncancel = trpc.stripe.uncancelSubscription.useMutation({
    onSuccess: async () => {
      await Promise.all([
        trpcUtils.workspace.getCurrent.invalidate(),
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
      ]);
      router.refresh();
      toast.info("API plan resumed");
    },
    onError: () =>
      toast.error("Failed to resume the plan. Please try again or contact support@unkey.com."),
  });

  return (
    <AlertBanner variant="warning">
      <TriangleWarning2 iconSize="md-regular" />
      <AlertBannerDescription>
        Your API plan ends on {formatDate(cancelAt)}. Afterwards your workspace will move to the
        free tier.
      </AlertBannerDescription>
      <AlertBannerActions>
        <AdminGate isAdmin={isAdmin}>
          {(disabled) => (
            <Button
              variant="outline"
              size="md"
              loading={uncancel.isLoading}
              disabled={disabled || uncancel.isLoading}
              onClick={() => uncancel.mutate()}
            >
              Resubscribe
            </Button>
          )}
        </AdminGate>
      </AlertBannerActions>
    </AlertBanner>
  );
}
