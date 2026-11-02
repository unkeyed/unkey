"use client";

import { useInvalidateWorkspaceQueries } from "@/hooks/use-invalidate-workspace-queries";
import type { SubscriptionProduct } from "@/lib/stripe/billingSubscriptions";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { toast } from "@unkey/ui";
import { useCallback, useEffect } from "react";

export function usePaymentIssues(): SubscriptionProduct[] {
  const { workspace } = useWorkspace();
  const { data } = trpc.stripe.getPaymentIssues.useQuery(undefined, {
    enabled: Boolean(workspace?.stripeCustomerId),
    staleTime: 5 * 60 * 1000,
  });
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
  const trpcUtils = trpc.useUtils();
  const mutation = trpc.stripe.getSubscriptionPaymentUrl.useMutation({
    onSuccess: ({ paymentUrl }) => window.location.assign(paymentUrl),
    onError: (err) => toast.error(err.message),
  });
  const { reset } = mutation;

  useEffect(() => {
    // Back from Stripe restores this page from the bfcache with the mutation
    // still marked successful, and the invoice may have been paid meanwhile.
    const onPageShow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        reset();
        trpcUtils.stripe.getPaymentIssues.invalidate();
      }
    };
    window.addEventListener("pageshow", onPageShow);
    return () => window.removeEventListener("pageshow", onPageShow);
  }, [reset, trpcUtils]);

  return {
    pay: (product: SubscriptionProduct) => mutation.mutate({ product }),
    // The page stays interactive between onSuccess and the navigation landing.
    busy: mutation.isLoading || mutation.isSuccess,
  };
}
