import { Code } from "@/components/ui/code";
import { getAuth as getBaseAuth } from "@/lib/auth/get-auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import Stripe from "stripe";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: {
    session_id?: string;
  };
};

/**
 * Enhanced auth check for payment flows that handles temporary session issues
 * and validates against Stripe session data as a fallback
 */
async function getAuthForPaymentFlow(stripeSessionId?: string) {
  // First attempt: try normal auth
  try {
    const authResult = await getBaseAuth();
    if (authResult.userId && authResult.orgId) {
      return authResult;
    }
  } catch (error) {
    console.warn("Primary auth check failed during payment flow:", error);
  }

  // If we have a Stripe session, we can validate against that
  if (stripeSessionId) {
    try {
      const e = stripeEnv();
      if (e) {
        const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
          apiVersion: "2023-10-16",
          typescript: true,
        });

        const session = await stripe.checkout.sessions.retrieve(
          stripeSessionId
        );
        if (session?.client_reference_id) {
          const stripeSessionReference = session.client_reference_id;
          // Find workspace by ID from Stripe session
          const workspace = await db.query.workspaces.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(
                eq(table.orgId, stripeSessionReference),
                isNull(table.deletedAtM)
              ),
          });

          if (workspace) {
            // Second attempt: retry auth after a brief delay
            await new Promise((resolve) => setTimeout(resolve, 1000));
            try {
              const retryAuthResult = await getBaseAuth();
              if (
                retryAuthResult.userId &&
                retryAuthResult.orgId === workspace.orgId
              ) {
                return retryAuthResult;
              }
            } catch (retryError) {
              console.warn("Retry auth check failed:", retryError);
            }

            // Third attempt: validate that the user has access to this workspace
            // by checking if they're a member of the organization
            try {
              const finalAuthResult = await getBaseAuth();
              if (finalAuthResult.userId) {
                // If we have a userId but no orgId, but the workspace exists and belongs to them,
                // this might be a temporary session issue
                const userHasAccessToWorkspace =
                  finalAuthResult.orgId === workspace.orgId;
                if (userHasAccessToWorkspace) {
                  return {
                    userId: finalAuthResult.userId,
                    orgId: workspace.orgId,
                    role: finalAuthResult.role,
                    impersonator: finalAuthResult.impersonator,
                  };
                }
              }
            } catch (finalError) {
              console.error("Final auth validation failed:", finalError);
            }
          }
        }
      }
    } catch (stripeError) {
      console.error("Stripe session validation failed:", stripeError);
    }
  }

  // If all attempts fail, return null to trigger auth redirect
  return null;
}

export default async function StripeRedirect(props: Props) {
  // For initial checkout creation (no session_id), use regular auth
  if (!props.searchParams.session_id) {
    const { orgId } = await getBaseAuth();
    if (!orgId) {
      redirect("/auth/sign-in");
    }

    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    });
    if (!ws) {
      return redirect("/new");
    }

    const e = stripeEnv();
    if (!e) {
      return (
        <Empty>
          <Empty.Title>Stripe is not configured</Empty.Title>
          <Empty.Description>
            If you are selfhosting Unkey, you need to configure Stripe in your
            environment variables.
          </Empty.Description>
        </Empty>
      );
    }

    const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
      apiVersion: "2023-10-16",
      typescript: true,
    });

    const baseUrl = process.env.VERCEL
      ? process.env.VERCEL_TARGET_ENV === "production"
        ? "https://app.unkey.com"
        : `https://${process.env.VERCEL_BRANCH_URL}`
      : "http://localhost:3000";

    const session = await stripe.checkout.sessions.create({
      client_reference_id: ws.id,
      billing_address_collection: "auto",
      mode: "setup",
      success_url: `${baseUrl}/settings/billing/stripe/checkout?session_id={CHECKOUT_SESSION_ID}`,
      currency: "USD",
      customer_creation: "always",
    });

    if (!session.url) {
      return (
        <Empty>
          <Empty.Title>Empty Session</Empty.Title>
          <Empty.Description>The Stripe session</Empty.Description>
          <Code>{session.id}</Code>
          <Empty.Description>
            you are trying to access does not exist. Please contact
            support@unkey.dev.
          </Empty.Description>
        </Empty>
      );
    }

    return redirect(session.url);
  }

  // For return from Stripe (has session_id), use enhanced auth check
  const authResult = await getAuthForPaymentFlow(props.searchParams.session_id);
  if (!authResult) {
    console.error("Auth failed for payment flow, redirecting to sign-in");
    redirect("/auth/sign-in");
  }

  const { orgId } = authResult;

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.orgId, orgId!!), isNull(table.deletedAtM)),
  });
  if (!ws) {
    return redirect("/new");
  }

  const e = stripeEnv();
  if (!e) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your
          environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const baseUrl = process.env.VERCEL
    ? process.env.VERCEL_TARGET_ENV === "production"
      ? "https://app.unkey.com"
      : `https://${process.env.VERCEL_BRANCH_URL}`
    : "http://localhost:3000";

  const session = await stripe.checkout.sessions.retrieve(
    props.searchParams.session_id
  );
  if (!session) {
    return (
      <Empty>
        <Empty.Title>Stripe session not found</Empty.Title>
        <Empty.Description>
          The Stripe session you are trying to access does not exist. Please
          contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }

  // Validate that the workspace from auth matches the one from Stripe session
  if (session.client_reference_id !== ws.id) {
    console.error(
      `Workspace mismatch: auth workspace ${ws.id} vs Stripe session workspace ${session.client_reference_id}`
    );
    redirect("/auth/sign-in");
  }

  const customer = await stripe.customers.retrieve(session.customer as string);
  if (!customer) {
    return (
      <Empty>
        <Empty.Title>Stripe customer not found</Empty.Title>
        <Empty.Description>The Stripe customer</Empty.Description>
        <Code>{session.customer as string}</Code>
        <Empty.Description>
          you are trying to access does not exist. Please contact
          support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }

  if (!session.setup_intent) {
    return (
      <Empty>
        <Empty.Title>Stripe setup intent not found</Empty.Title>
        <Empty.Description>Stripe did not return a</Empty.Description>
        <Code>setup_intent</Code>
        <Empty.Description>
          id. Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }
  const setupIntent = await stripe.setupIntents.retrieve(
    session.setup_intent.toString()
  );
  if (!setupIntent.payment_method) {
    return (
      <Empty>
        <Empty.Title>Payment method not found</Empty.Title>
        <Empty.Description>
          Stripe did not return a valid payment method. Please contact
          support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }
  await stripe.customers.update(customer.id, {
    invoice_settings: {
      default_payment_method: setupIntent.payment_method.toString(),
    },
  });

  try {
    await db
      .update(schema.workspaces)
      .set({
        stripeCustomerId: customer.id,
      })
      .where(eq(schema.workspaces.id, ws.id));
  } catch {
    return (
      <Empty>
        <Empty.Title>Failed to update workspace</Empty.Title>
        <Empty.Description>
          There was an error updating your workspace with payment information.
          Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }
  return redirect(`${baseUrl}/settings/billing`);
}
