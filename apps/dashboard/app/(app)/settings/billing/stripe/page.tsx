import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Stripe from "stripe";

type Props = {
  searchParams: {
    action:
      | "portal"
      | "start_trial"
      | "payment_intent"
      | "subscription_update"
      | "subscription_cancel";
    session_id?: string;
  };
};

export default async function StripeRedirect(props: Props) {
  const tenantId = getTenantId();
  if (!tenantId) {
    return redirect("/auth/sign-in");
  }

  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      auditLogBuckets: {
        where: (table, { eq }) => eq(table.name, "unkey_mutations"),
      },
    },
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
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </Empty.Description>
      </Empty>
    );
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const baseUrl = process.env.VERCEL_URL
    ? process.env.VERCEL_TARGET_ENV === "production"
      ? "https://app.unkey.com"
      : `https://${process.env.URL}`
    : "http://localhost:3000";
  const returnUrl = headers().get("referer") ?? "https://app.unkey.com";

  switch (props.searchParams.action) {
    case "portal": {
      if (!ws.stripeCustomerId) {
        return (
          <Empty>
            <Empty.Title>No customer found</Empty.Title>
            <Empty.Description>Your workspace</Empty.Description>
            <Code>{ws.id}</Code>
            <Empty.Description>
              is not in Stripe yet. Please contact support@unkey.dev.
            </Empty.Description>
          </Empty>
        );
      }

      const { url } = await stripe.billingPortal.sessions.create({
        customer: ws.stripeCustomerId,
        return_url: returnUrl,
      });
      return redirect(url);
    }
    case "start_trial": {
      const session = await stripe.checkout.sessions.retrieve(props.searchParams.session_id!);
      if (!session) {
        return (
          <Empty>
            <Empty.Title>Stripe session not found</Empty.Title>
            <Empty.Description>The Stripe session</Empty.Description>
            <Code>{props.searchParams.session_id}</Code>
            <Empty.Description>
              you are trying to access does not exist. Please contact support@unkey.dev.
            </Empty.Description>
          </Empty>
        );
      }
      const customer = await stripe.customers.retrieve(session.customer as string);
      if (!customer) {
        return (
          <Empty>
            <Empty.Title>Stripe customer not found</Empty.Title>
            <Empty.Description>The Stripe customer</Empty.Description>
            <Code>{session.customer as string}</Code>
            <Empty.Description>
              you are trying to access does not exist. Please contact support@unkey.dev.
            </Empty.Description>
          </Empty>
        );
      }

      await db
        .update(schema.workspaces)
        .set({
          stripeCustomerId: customer.id,
        })
        .where(eq(schema.workspaces.id, ws.id));
      const sub = await stripe.subscriptions.create({
        customer: customer.id,
        items: [
          {
            price: e.STRIPE_TRIAL_PRICE_ID,
          },
        ],
        billing_cycle_anchor_config: {
          day_of_month: 1,
        },

        proration_behavior: "create_prorations",
        trial_period_days: 14,
        trial_settings: {
          end_behavior: {
            missing_payment_method: "cancel",
          },
        },
      });
      await db
        .update(schema.workspaces)
        .set({
          stripeSubscriptionId: sub.id,
        })
        .where(eq(schema.workspaces.id, ws.id));

      return redirect(`${baseUrl}/settings/billing`);
    }
    case "payment_intent": {
      const session = await stripe.checkout.sessions.create({
        client_reference_id: ws.id,
        billing_address_collection: "auto",
        mode: "setup",
        success_url: `${baseUrl}/settings/billing/stripe?action=start_trial&session_id={CHECKOUT_SESSION_ID}`,
        currency: "USD",
        customer_creation: "always",
      });

      if (!session.url) {
        return (
          <Empty>
            <Empty.Title>Stripe was unable to generate a session</Empty.Title>
            <Empty.Description>The Stripe session</Empty.Description>
            <Code>{session.id}</Code>
            <Empty.Description>
              you are trying to access does not exist. Please contact support@unkey.dev.
            </Empty.Description>
          </Empty>
        );
      }

      return redirect(session.url);
    }
    case "subscription_update": {
      const { url } = await stripe.billingPortal.sessions.create({
        customer: ws.stripeCustomerId!,
        return_url: returnUrl,
        flow_data: {
          type: props.searchParams.action,
          subscription_update: {
            subscription: ws.stripeSubscriptionId!,
          },
          after_completion: {
            type: "redirect",
            redirect: {
              return_url: returnUrl,
            },
          },
        },
      });
      return redirect(url);
    }

    case "subscription_cancel": {
      const { url } = await stripe.billingPortal.sessions.create({
        customer: ws.stripeCustomerId!,
        return_url: returnUrl,
        flow_data: {
          type: props.searchParams.action,
          subscription_cancel: {
            subscription: ws.stripeSubscriptionId!,
          },

          after_completion: {
            type: "redirect",
            redirect: {
              return_url: returnUrl,
            },
          },
        },
      });
      return redirect(url);
    }
  }

  return (
    <Empty>
      <Empty.Title>Stripe Error</Empty.Title>
      <Empty.Description>
        You should have been redirected, please report this to support@unkey.dev
      </Empty.Description>
    </Empty>
  );
}
