"use client";

import { useInvalidateWorkspaceQueries } from "@/hooks/use-invalidate-workspace-queries";
import { formatDate } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
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

export function BillingNotices({
  isAdmin,
  subscription,
}: {
  isAdmin: boolean | undefined;
  subscription?: BillingInfo["subscription"];
}) {
  return (
    <>
      {subscription?.status === "incomplete_expired" ? <PaymentExpired /> : null}
      {subscription?.cancelAt && subscription.cancelAt > Date.now() ? (
        <ScheduledCancellation isAdmin={isAdmin} cancelAt={subscription.cancelAt} />
      ) : null}
    </>
  );
}

function PaymentExpired() {
  return (
    <AlertBanner variant="error">
      <IconTriangleWarningOutline18 className="size-3.5" />
      <AlertBannerTitle>Payment required</AlertBannerTitle>
      <AlertBannerDescription>
        The previous payment attempt expired. Choose your plan again to retry.
      </AlertBannerDescription>
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
  const invalidateWorkspace = useInvalidateWorkspaceQueries();

  const uncancel = trpc.stripe.uncancelSubscription.useMutation({
    onSuccess: async () => {
      await Promise.all([
        invalidateWorkspace(),
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
      <IconTriangleWarningOutline18 className="size-3.5" />
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
