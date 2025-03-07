import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Empty } from "@unkey/ui";
import { redirect } from "next/navigation";
import Stripe from "stripe";

type Props = {
  searchParams: {
    session_id?: string;
    product_id?: string;
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
      : `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  if (!props.searchParams.session_id) {
    return (
      <Empty>
        <Empty.Title>Missing session_id</Empty.Title>
        <Empty.Description>You need to provide a </Empty.Description>
        <Code>session_id</Code>
        <Empty.Description>as query parameter. Please contact support@unkey.dev.</Empty.Description>
      </Empty>
    );
  }

  const session = await stripe.checkout.sessions.retrieve(props.searchParams.session_id);
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

  if (!session.setup_intent) {
    return (
      <Empty>
        <Empty.Title>Stripe setup intent not found</Empty.Title>
        <Empty.Description>Stripe did not return a</Empty.Description>
        <Code>setup_intent</Code>
        <Empty.Description>id. Please contact support@unkey.dev.</Empty.Description>
      </Empty>
    );
  }
  const setupIntent = await stripe.setupIntents.retrieve(session.setup_intent.toString());
  if (!setupIntent.payment_method) {
    return (
      <Empty>
        <Empty.Title>Payment method not found</Empty.Title>
        <Empty.Description>
          Stripe did not return a valid payment method. Please contact support@unkey.dev.
        </Empty.Description>
      </Empty>
    );
  }
  await stripe.customers.update(customer.id, {
    invoice_settings: {
      default_payment_method: setupIntent.payment_method.toString(),
    },
  });

  const productId = props.searchParams.product_id ?? e.STRIPE_PRODUCT_IDS_PRO[0];
  const product = await stripe.products.retrieve(productId);

  if (!product) {
    return (
      <Empty>
        <Empty.Title>Stripe product</Empty.Title>
        <Empty.Description>Stripe did not find the product</Empty.Description>
        <Code>{productId}</Code>
        <Empty.Description>. Please contact support@unkey.dev.</Empty.Description>
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
        price: product.default_price!.toString(),
      },
    ],
    billing_cycle_anchor_config: {
      day_of_month: 1,
    },

    proration_behavior: "always_invoice",
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
  await db
    .insert(schema.quotas)
    .values({
      workspaceId: ws.id,
      requestsPerMonth: Number.parseInt(product.metadata.quota_requests_per_month),
      logsRetentionDays: Number.parseInt(product.metadata.quota_logs_retention_days),
      auditLogsRetentionDays: Number.parseInt(product.metadata.quota_audit_logs_retention_days),
      team: true,
    })
    .onDuplicateKeyUpdate({
      set: {
        requestsPerMonth: Number.parseInt(product.metadata.quota_requests_per_month),
        logsRetentionDays: Number.parseInt(product.metadata.quota_logs_retention_days),
        auditLogsRetentionDays: Number.parseInt(product.metadata.quota_audit_logs_retention_days),
        team: true,
      },
    });

  return redirect(`${baseUrl}/settings/billing`);
}
