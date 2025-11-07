"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
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

  useEffect(() => {
    // Track if component is still mounted to prevent state updates after unmount
    let isMounted = true;

    if (!sessionId) {
      setProcessedData({});
      setLoading(false);
      return;
    }

    const processStripeSession = async (
      updateCustomerFn: typeof updateCustomerMutation.mutateAsync,
      updateWorkspaceFn: typeof updateWorkspaceStripeCustomerMutation.mutateAsync,
    ) => {
      try {
        if (!isMounted) {
          return;
        }
        setLoading(true);

        // Get checkout session
        const sessionResponse = await trpcUtils.stripe.getCheckoutSession.fetch({
          sessionId: sessionId,
        });

        if (!sessionResponse) {
          console.warn("Stripe session not found");
          if (!isMounted) {
            return;
          }
          setProcessedData({});
          setLoading(false);
          return;
        }

        const workspaceId = sessionResponse.client_reference_id;
        if (!workspaceId) {
          console.warn("Stripe session client_reference_id not found");
          if (!isMounted) {
            return;
          }
          setProcessedData({});
          setLoading(false);
          return;
        }

        // Get workspace details to get the slug
        const workspace = await trpcUtils.workspace.getById.fetch({
          workspaceId: workspaceId,
        });

        if (!isMounted) {
          return;
        }

        // Check if we have customer and setup intent
        if (!sessionResponse.customer || !sessionResponse.setup_intent) {
          console.warn("Stripe customer or setup intent not found");
          if (!isMounted) {
            return;
          }
          setProcessedData({ workspaceSlug: workspace.slug });
          setLoading(false);
          return;
        }

        // Get customer details
        const customer = await trpcUtils.stripe.getCustomer.fetch({
          customerId: sessionResponse.customer,
        });

        if (!isMounted) {
          return;
        }

        // Get setup intent details
        const setupIntent = await trpcUtils.stripe.getSetupIntent.fetch({
          setupIntentId: sessionResponse.setup_intent,
        });

        if (!isMounted) {
          return;
        }

        if (!customer || !setupIntent?.payment_method) {
          console.warn("Customer or payment method not found");
          if (!isMounted) {
            return;
          }
          setProcessedData({ workspaceSlug: workspace.slug });
          setLoading(false);
          return;
        }

        // Update customer with default payment method
        try {
          await updateCustomerFn({
            customerId: customer.id,
            paymentMethod: setupIntent.payment_method,
          });

          if (!isMounted) {
            return;
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : "Unknown error";
          console.error("Failed to update customer with payment method:", {
            error: errorMessage,
            customerId: "redacted", // Don't log PII
            hasPaymentMethod: !!setupIntent.payment_method,
          });
          if (!isMounted) {
            return;
          }
          setError(`Failed to set up payment method: ${errorMessage}`);
          setLoading(false);
          return;
        }

        // Update workspace with stripe customer ID
        try {
          await updateWorkspaceFn({
            stripeCustomerId: customer.id,
          });

          if (!isMounted) {
            return;
          }

          await trpcUtils.workspace.invalidate();
          await trpcUtils.stripe.invalidate();
          await trpcUtils.billing.invalidate();
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : "Unknown error";
          console.error("Failed to update workspace with payment method:", { error: errorMessage });
          if (!isMounted) {
            return;
          }
          setError("Failed to update workspace with payment information");
          setLoading(false);
          return;
        }

        // Check if this is a first-time user by getting billing info
        try {
          const billingInfo = await trpcUtils.stripe.getBillingInfo.fetch();

          if (!isMounted) {
            return;
          }

          const isFirstTimeUser = !billingInfo.hasPreviousSubscriptions;

          if (isFirstTimeUser) {
            // Get products for plan selection
            try {
              const products = await trpcUtils.stripe.getProducts.fetch();

              if (!isMounted) {
                return;
              }

              setProcessedData({
                workspaceSlug: workspace.slug,
                showPlanSelection: true,
                products: products,
              });
            } catch (error) {
              console.error("Failed to load products:", error);
              // Fall back to regular billing page if products fail to load
              if (!isMounted) {
                return;
              }
              setProcessedData({ workspaceSlug: workspace.slug });
            }
          } else {
            if (!isMounted) {
              return;
            }
            setProcessedData({ workspaceSlug: workspace.slug });
          }
        } catch (error) {
          console.error("Failed to get billing info:", error);
          // Fall back to regular billing page
          if (!isMounted) {
            return;
          }
          setProcessedData({ workspaceSlug: workspace.slug });
        }

        if (!isMounted) {
          return;
        }
        setLoading(false);
      } catch (error) {
        console.error("Error processing Stripe session:", error);
        if (!isMounted) {
          return;
        }
        setError("Failed to process payment session");
        setLoading(false);
      }
    };

    processStripeSession(
      updateCustomerMutation.mutateAsync,
      updateWorkspaceStripeCustomerMutation.mutateAsync,
    );

    // Cleanup function to prevent state updates after unmount
    return () => {
      isMounted = false;
    };
  }, [
    sessionId,
    trpcUtils,
    updateCustomerMutation.mutateAsync,
    updateWorkspaceStripeCustomerMutation.mutateAsync,
  ]);

  if (loading) {
    return <PageLoading message="Processing payment..." />;
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
    <Suspense fallback={<PageLoading message="Loading..." />}>
      <SuccessContent />
    </Suspense>
  );
}
