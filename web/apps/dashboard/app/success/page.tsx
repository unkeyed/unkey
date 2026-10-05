"use client";

import { PageLoading } from "@/components/dashboard/page-loading";
import type { CheckoutOutcome } from "@/lib/billing/upgrade-result";
import { DEPLOY_PLANS } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
} from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { SuccessClient } from "./client";

const SUPPORT_SUFFIX = "Please contact support@unkey.com if this issue persists.";

const endWithPunctuation = (message: string): string => {
  return /[.!?]$/.test(message) ? message : `${message}.`;
};

// Prepends the failed billing step, since server messages are phrased without
// knowing the caller ("Workspace not found.") and do not say which step broke.
const toUserFacingError = (error: unknown, context: string): string => {
  const detail = error instanceof Error ? error.message : "Unknown error";

  return `${context}: ${endWithPunctuation(detail)} ${SUPPORT_SUFFIX}`;
};

type ProcessedData = {
  workspaceSlug?: string;
  outcome?: CheckoutOutcome;
  showPlanSelection?: boolean;
};

function SuccessContent() {
  const searchParams = useSearchParams();
  const sessionId = searchParams?.get("session_id") ?? null;
  const intent = searchParams?.get("intent") ?? null;
  const plan = searchParams?.get("plan") ?? null;
  const from = searchParams?.get("from") ?? null;
  const returnTo = searchParams?.get("returnTo") ?? null;

  const [processedData, setProcessedData] = useState<ProcessedData>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const updateCustomerMutation = trpc.stripe.updateCustomer.useMutation();
  const updateWorkspaceStripeCustomerMutation =
    trpc.stripe.updateWorkspaceStripeCustomer.useMutation();
  const linkApiSubscriptionMutation = trpc.stripe.linkApiSubscription.useMutation();
  const linkDeploySubscriptionMutation = trpc.stripe.linkDeploySubscription.useMutation();
  const subscribeDeployMutation = trpc.stripe.subscribeDeploy.useMutation();

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
      linkApiFn: typeof linkApiSubscriptionMutation.mutateAsync,
      linkDeployFn: typeof linkDeploySubscriptionMutation.mutateAsync,
      subscribeDeployFn: typeof subscribeDeployMutation.mutateAsync,
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
        const workspace = await trpcUtils.workspace.getById.fetch();

        if (!isMounted) {
          return;
        }

        // API subscription Checkout owns the first payment, including CVC
        // recollection and 3DS. Link the paid subscription before returning to
        // billing; the completed webhook races through the same idempotent path
        // if the user closes this page early.
        if (intent === "api-subscription" && sessionResponse.subscription) {
          try {
            await linkApiFn({ sessionId });
          } catch (error) {
            const entitled = await trpcUtils.stripe.getBillingInfo
              .fetch(undefined, { staleTime: 0 })
              .then((billing) => Boolean(billing.currentProductId))
              .catch(() => false);
            if (!isMounted) {
              return;
            }
            if (!entitled) {
              const errorMessage = error instanceof Error ? error.message : "Unknown error";
              setError(`Failed to activate your API plan: ${errorMessage}`);
              setLoading(false);
              return;
            }
          }

          if (!isMounted) {
            return;
          }
          await trpcUtils.workspace.invalidate();
          await trpcUtils.stripe.invalidate();
          await trpcUtils.billing.invalidate();
          setProcessedData({ workspaceSlug: workspace.slug, outcome: "subscribed" });
          setLoading(false);
          return;
        }

        // Subscription-mode deploy checkout: Stripe already created and charged
        // the subscription, so there is no setup intent to process.
        // The checkout.session.completed webhook may have linked it already; the
        // shared linker is idempotent, so this is a safe fast-path.
        if (
          intent === "deploy" &&
          (sessionResponse.mode === "subscription" || sessionResponse.subscription)
        ) {
          try {
            await linkDeployFn({ sessionId });
          } catch (error) {
            // This mutation is only a fast-path; the checkout.session.completed
            // webhook is the guaranteed linker. If it already linked (or a
            // transient error hit after the write committed), the workspace is
            // entitled — treat that as success rather than showing an alarming
            // error for a charge that actually went through. Mirrors the
            // entitlement-first check in projects/page.tsx usePendingSubscribe.
            const entitled = await trpcUtils.stripe.getDeployEntitlement
              .fetch(undefined, { staleTime: 0 })
              .then((e) => Boolean(e?.entitled))
              .catch(() => false);
            if (!isMounted) {
              return;
            }
            if (!entitled) {
              setError(toUserFacingError(error, "Failed to activate your Compute plan"));
              setLoading(false);
              return;
            }
            // Entitled despite the fast-path error — fall through to success.
          }

          if (!isMounted) {
            return;
          }

          await trpcUtils.workspace.invalidate();
          await trpcUtils.stripe.invalidate();
          await trpcUtils.billing.invalidate();

          setProcessedData({ workspaceSlug: workspace.slug, outcome: "subscribed" });
          setLoading(false);
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

        // Pass sessionId so the server can verify the setup intent belongs to
        // a session bound to this workspace.
        const setupIntent = await trpcUtils.stripe.getSetupIntent.fetch({
          setupIntentId: sessionResponse.setup_intent,
          sessionId,
        });

        if (!isMounted) {
          return;
        }

        if (!setupIntent?.payment_method) {
          console.warn("Payment method not found");
          if (!isMounted) {
            return;
          }
          setProcessedData({ workspaceSlug: workspace.slug });
          setLoading(false);
          return;
        }

        try {
          await updateCustomerFn({
            sessionId,
            paymentMethod: setupIntent.payment_method,
          });

          if (!isMounted) {
            return;
          }
        } catch (error) {
          console.error("Failed to update customer with payment method:", {
            error: error instanceof Error ? error.message : "Unknown error",
            hasPaymentMethod: !!setupIntent.payment_method,
          });
          if (!isMounted) {
            return;
          }
          setError(toUserFacingError(error, "Failed to set up the payment method"));
          setLoading(false);
          return;
        }

        // Update workspace with stripe customer ID. The mutation resolves the
        // customer id from the checkout session server-side and verifies the
        // session belongs to this workspace, so we pass sessionId instead of
        // a client-supplied stripeCustomerId.
        try {
          await updateWorkspaceFn({
            sessionId,
          });

          if (!isMounted) {
            return;
          }

          await trpcUtils.workspace.invalidate();
          await trpcUtils.stripe.invalidate();
          await trpcUtils.billing.invalidate();
        } catch (error) {
          console.error("Failed to update workspace with payment method:", {
            error: error instanceof Error ? error.message : "Unknown error",
          });
          if (!isMounted) {
            return;
          }
          setError(toUserFacingError(error, "Failed to update workspace with payment information"));
          setLoading(false);
          return;
        }

        // Setup mode only saved the card. A failed subscribe falls through to the
        // projects hand-off, which retries and owns the decline recovery.
        const deployPlan =
          intent === "deploy" ? DEPLOY_PLANS.find((known) => known === plan) : undefined;
        if (deployPlan) {
          const subscribed = await subscribeDeployFn({ plan: deployPlan })
            .then(() => true)
            .catch((error) => {
              console.error("Compute subscribe failed after setup checkout", {
                plan: deployPlan,
                error: error instanceof Error ? error.message : error,
              });
              return false;
            });
          if (!isMounted) {
            return;
          }
          if (subscribed) {
            await trpcUtils.stripe.invalidate();
            await trpcUtils.workspace.invalidate();
            setProcessedData({ workspaceSlug: workspace.slug, outcome: "subscribed" });
            setLoading(false);
            return;
          }
        }

        // Check if this is a first-time user by getting billing info
        try {
          const billingInfo = await trpcUtils.stripe.getBillingInfo.fetch();

          if (!isMounted) {
            return;
          }

          const isFirstTimeUser = !billingInfo.hasPreviousSubscriptions;

          if (isFirstTimeUser && !intent) {
            if (billingInfo.products && billingInfo.products.length > 0) {
              setProcessedData({
                workspaceSlug: workspace.slug,
                showPlanSelection: true,
              });
            } else {
              // Fall back to regular billing page if products are empty or undefined
              setProcessedData({ workspaceSlug: workspace.slug });
            }
          } else {
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
        setError(toUserFacingError(error, "Failed to process payment session"));
        setLoading(false);
      }
    };

    processStripeSession(
      updateCustomerMutation.mutateAsync,
      updateWorkspaceStripeCustomerMutation.mutateAsync,
      linkApiSubscriptionMutation.mutateAsync,
      linkDeploySubscriptionMutation.mutateAsync,
      subscribeDeployMutation.mutateAsync,
    );

    // Cleanup function to prevent state updates after unmount
    return () => {
      isMounted = false;
    };
  }, [
    sessionId,
    intent,
    trpcUtils,
    updateCustomerMutation.mutateAsync,
    updateWorkspaceStripeCustomerMutation.mutateAsync,
    linkApiSubscriptionMutation.mutateAsync,
    linkDeploySubscriptionMutation.mutateAsync,
    subscribeDeployMutation.mutateAsync,
    plan,
  ]);

  if (loading) {
    return <PageLoading message="Processing payment..." />;
  }

  if (error) {
    return (
      <PageContainer>
        <PageBody>
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>Payment Processing Error</EmptyStateTitle>
              <EmptyStateDescription>{error}</EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        </PageBody>
      </PageContainer>
    );
  }

  return (
    <SuccessClient
      workSpaceSlug={processedData.workspaceSlug}
      outcome={processedData.outcome ?? "none"}
      showPlanSelection={processedData.showPlanSelection}
      intent={intent ?? undefined}
      plan={plan ?? undefined}
      from={from ?? undefined}
      returnTo={returnTo ?? undefined}
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
