import { eq, mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { Stripe } from "stripe";
async function main() {
  const stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const workspaceId = "ws_wB4SmWrYkhSbWE2rH61S6gMseWw";
  const productId = "prod_Rtu3rLbjwprz7p";

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.id, workspaceId),
  });

  if (!workspace) {
    throw new Error("workspace not found");
  }

  const product = await stripe.products.retrieve(productId);
  if (!product) {
    throw new Error("product not found");
  }

  if (!workspace.stripeCustomerId) {
    throw new Error("missing stripeCustomerId");
  }
  if (workspace.stripeSubscriptionId) {
    throw new Error("workspaces already has a subscription");
  }

  const customer = await stripe.customers.retrieve(workspace.stripeCustomerId);
  if (!customer) {
    throw new Error("customer not found");
  }

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
  });
  await db
    .update(schema.workspaces)
    .set({
      stripeSubscriptionId: sub.id,
      tier: product.name,
      subscriptions: {},
    })
    .where(eq(schema.workspaces.id, workspace.id));
  await db
    .insert(schema.quotas)
    .values({
      workspaceId: workspace.id,
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

  await conn.end();
}

main();
