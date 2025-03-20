import { insertAuditLogs } from "@/lib/audit";
import { type Quotas, db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import Stripe from "stripe";

export const runtime = "nodejs";

export const POST = async (req: Request): Promise<Response> => {
  const signature = req.headers.get("stripe-signature");
  if (!signature) {
    throw new Error("Signature missing");
  }

  const e = stripeEnv();

  if (!e) {
    throw new Error("stripe env variables are not set up");
  }

  const stripe = new Stripe(stripeEnv()!.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  const event = stripe.webhooks.constructEvent(
    await req.text(),
    signature,
    e.STRIPE_WEBHOOK_SECRET,
  );

  switch (event.type) {
    case "customer.subscription.deleted": {
      const sub = event.data.object as Stripe.Subscription;

      const ws = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.stripeSubscriptionId, sub.id), isNull(table.deletedAtM)),
        with: {
          auditLogBuckets: {
            where: (table, { eq }) => eq(table.name, "unkey_mutations"),
          },
        },
      });
      if (!ws) {
        return new Response("workspace does not exist", { status: 500 });
      }
      await db
        .update(schema.workspaces)
        .set({
          stripeSubscriptionId: null,
        })
        .where(eq(schema.workspaces.id, ws.id));

      const freeTierQuotas: Omit<Quotas, "workspaceId"> = {
        requestsPerMonth: 150_000,
        logsRetentionDays: 7,
        auditLogsRetentionDays: 30,
        team: false,
      };
      await db
        .insert(schema.quotas)
        .values({
          workspaceId: ws.id,
          ...freeTierQuotas,
        })
        .onDuplicateKeyUpdate({
          set: freeTierQuotas,
        });

      await insertAuditLogs(db, ws.auditLogBuckets[0].id, {
        workspaceId: ws.id,
        actor: {
          type: "system",
          id: "stripe",
        },
        event: "workspace.update",
        description: "Cancelled subscription.",
        resources: [],
        context: {
          location: "",
          userAgent: undefined,
        },
      });
      break;
    }

    default:
      console.error("Incoming stripe event, that should not be received", event.type);
      break;
  }
  return new Response("OK");
};
