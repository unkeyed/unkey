"use client";

import { useInvalidateWorkspaceQueries } from "@/hooks/use-invalidate-workspace-queries";
import type { SubscriptionProduct } from "@/lib/stripe/billingSubscriptions";
import { SUBSCRIPTION_STATUS_TTL_MS } from "@/lib/stripe/subscriptionUtils";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { toast } from "@unkey/ui";
import { useCallback, useEffect, useSyncExternalStore } from "react";

// Back from Stripe restores the page from the bfcache with the old React state:
// pay buttons still busy and payment issues that may have been paid meanwhile.
const pageRestore = {
  leavingForInvoice: false,
  restores: 0,
  subscribers: new Set<() => void>(),
};

function onPageShow(event: PageTransitionEvent) {
  if (!event.persisted) {
    return;
  }
  pageRestore.leavingForInvoice = false;
  pageRestore.restores += 1;
  notifyPageRestore();
}

function notifyPageRestore() {
  for (const subscriber of pageRestore.subscribers) {
    subscriber();
  }
}

function subscribePageRestore(subscriber: () => void) {
  if (pageRestore.subscribers.size === 0) {
    window.addEventListener("pageshow", onPageShow);
  }
  pageRestore.subscribers.add(subscriber);
  return () => {
    pageRestore.subscribers.delete(subscriber);
    if (pageRestore.subscribers.size === 0) {
      window.removeEventListener("pageshow", onPageShow);
    }
  };
}

export function usePaymentIssues(): SubscriptionProduct[] {
  const { workspace } = useWorkspace();
  const { data, refetch } = trpc.stripe.getPaymentIssues.useQuery(undefined, {
    enabled: Boolean(workspace?.stripeCustomerId),
    staleTime: SUBSCRIPTION_STATUS_TTL_MS,
  });
  const restores = useSyncExternalStore(
    subscribePageRestore,
    () => pageRestore.restores,
    () => 0,
  );

  useEffect(() => {
    if (restores > 0) {
      refetch();
    }
  }, [restores, refetch]);

  return data ?? [];
}

export function useInvalidateBillingQueries() {
  const trpcUtils = trpc.useUtils();
  const invalidateWorkspace = useInvalidateWorkspaceQueries();

  return useCallback(
    () =>
      Promise.all([
        invalidateWorkspace(),
        trpcUtils.stripe.getBillingInfo.invalidate(),
        trpcUtils.stripe.getUpcomingInvoice.invalidate(),
        trpcUtils.stripe.getPaymentIssues.invalidate(),
      ]),
    [trpcUtils, invalidateWorkspace],
  );
}

export function usePayInvoice() {
  const mutation = trpc.stripe.getSubscriptionPaymentUrl.useMutation({
    onSuccess: ({ paymentUrl }) => {
      pageRestore.leavingForInvoice = true;
      notifyPageRestore();
      window.location.assign(paymentUrl);
    },
    onError: (err) => toast.error(err.message),
  });
  const leavingForInvoice = useSyncExternalStore(
    subscribePageRestore,
    () => pageRestore.leavingForInvoice,
    () => false,
  );

  return {
    pay: (product: SubscriptionProduct) => mutation.mutate({ product }),
    busy: mutation.isLoading || leavingForInvoice,
  };
}
