"use client";

import { trpc } from "@/lib/trpc/client";
import { Empty, Loading } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";
import { SuccessClient } from "./client";

type ProcessedData = {
  workspaceSlug?: string;
  showPlanSelection?: boolean;
  products?: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quotas: {
      requestsPerMonth: number;
    };
  }>;
};

function SuccessContent() {
  const searchParams = useSearchParams();
  const sessionId = searchParams?.get("session_id") ?? null;

  const [processedData, setProcessedData] = useState<ProcessedData>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const updateCustomerMutation = trpc.stripe.updateCustomer.useMutation();
  const updateWorkspaceStripeCustomerMutation =
    trpc.stripe.updateWorkspaceStripeCustomer.useMutation();

  const trpcUtils = trpc.useUtils();

  const updateCustomer = useCallback(
    (data: { customerId: string; paymentMethod: string }) =>
      updateCustomerMutation.mutateAsync(data),
    [updateCustomerMutation.mutateAsync],
  );

  const updateWorkspaceStripeCustomer = useCallback(
    (data: { stripeCustomerId: string }) => updateWorkspaceStripeCustomerMutation.mutateAsync(data),
    [updateWorkspaceStripeCustomerMutation.mutateAsync],
  );

  useEffect(() => {
    if (!sessionId) {
      setProcessedData({});
      setLoading(false);
      return;
    }

    const processStripeSession = async () => {
      try {
        setLoading(true);

        // Get checkout session
        const sessionResponse = await trpcUtils.stripe.getCheckoutSession.fetch({
          sessionId: sessionId,
        });

        if (!sessionResponse) {
          console.warn("Stripe session not found");
          setProcessedData({});
          setLoading(false);
          return;
        }

        const workspaceId = sessionResponse.client_reference_id;
        if (!workspaceId) {
          console.warn("Stripe session client_reference_id not found");
          setProcessedData({});
          setLoading(false);
          return;
        }

        // Get workspace details to get the slug
        const workspace = await trpcUtils.workspace.getById.fetch({
          workspaceId: workspaceId,
        });

        // Check if we have customer and setup intent
        if (!sessionResponse.customer || !sessionResponse.setup_intent) {
          console.warn("Stripe customer or setup intent not found");
          setProcessedData({ workspaceSlug: workspace.slug });
          setLoading(false);
          return;
        }

        // Get customer details
        const customer = await trpcUtils.stripe.getCustomer.fetch({
          customerId: sessionResponse.customer,
        });

        // Get setup intent details
        const setupIntent = await trpcUtils.stripe.getSetupIntent.fetch({
          setupIntentId: sessionResponse.setup_intent,
        });

        if (!customer || !setupIntent?.payment_method) {
          console.warn("Customer or payment method not found");
          setProcessedData({ workspaceSlug: workspace.slug });
          setLoading(false);
          return;
        }

        // Update customer with default payment method
        try {
          await updateCustomer({
            customerId: customer.id,
            paymentMethod: setupIntent.payment_method,
          });
        } catch (error) {
          console.error("Failed to update customer:", error);
          // Continue processing even if customer update fails
        }

        // Update workspace with stripe customer ID
        try {
          await updateWorkspaceStripeCustomer({
            stripeCustomerId: customer.id,
          });
          await trpcUtils.workspace.invalidate();
        } catch (error) {
          console.error("Failed to update workspace:", error);
          setError("Failed to update workspace with payment information");
          setLoading(false);
          return;
        }

        // Check if this is a first-time user by getting billing info
        try {
          const billingInfo = await trpcUtils.stripe.getBillingInfo.fetch();
          const isFirstTimeUser = !billingInfo.hasPreviousSubscriptions;

          if (isFirstTimeUser) {
            // Get products for plan selection
            try {
              const products = await trpcUtils.stripe.getProducts.fetch();
              setProcessedData({
                workspaceSlug: workspace.slug,
                showPlanSelection: true,
                products: products,
              });
            } catch (error) {
              console.error("Failed to load products:", error);
              // Fall back to regular billing page if products fail to load
              setProcessedData({ workspaceSlug: workspace.slug });
            }
          } else {
            setProcessedData({ workspaceSlug: workspace.slug });
          }
        } catch (error) {
          console.error("Failed to get billing info:", error);
          // Fall back to regular billing page
          setProcessedData({ workspaceSlug: workspace.slug });
        }

        setLoading(false);
      } catch (error) {
        console.error("Error processing Stripe session:", error);
        setError("Failed to process payment session");
        setLoading(false);
      }
    };

    processStripeSession();
  }, [sessionId, trpcUtils, updateCustomer, updateWorkspaceStripeCustomer]);

  if (loading) {
    return (
      <Empty className="flex items-center justify-center h-screen w-full">
        <Loading type="spinner" size={40} />
      </Empty>
    );
  }

  if (error) {
    return (
      <Empty>
        <Empty.Title>Payment Processing Error</Empty.Title>
        <Empty.Description>
          {error}. Please contact support@unkey.dev if this issue persists.
        </Empty.Description>
      </Empty>
    );
  }

  return (
    <SuccessClient
      workSpaceSlug={processedData.workspaceSlug}
      showPlanSelection={processedData.showPlanSelection}
      products={processedData.products}
    />
  );
}

export default function SuccessPage() {
  return (
    <Suspense
      fallback={
        <Empty className="flex items-center justify-center h-screen w-full">
          <Loading type="spinner" size={40} />
        </Empty>
      }
    >
      <SuccessContent />
    </Suspense>
  );
}
