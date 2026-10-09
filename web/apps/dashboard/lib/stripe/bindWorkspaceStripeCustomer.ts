import { insertAuditLogs } from "@/lib/audit";
import { db, schema, sql, transactionWithRetry } from "@/lib/db";

export async function bindWorkspaceStripeCustomer(input: {
  workspaceId: string;
  stripeCustomerId: string;
  userId: string;
}): Promise<void> {
  await transactionWithRetry(db, async (tx) => {
    await tx
      .insert(schema.workspaceBilling)
      .values({ workspaceId: input.workspaceId, stripeCustomerId: input.stripeCustomerId })
      .onDuplicateKeyUpdate({
        set: {
          stripeCustomerId: sql`COALESCE(${schema.workspaceBilling.stripeCustomerId}, ${input.stripeCustomerId})`,
        },
      });
    await insertAuditLogs(tx, {
      workspaceId: input.workspaceId,
      actor: { type: "user", id: input.userId },
      event: "workspace.update",
      description: "Updated Stripe customer ID",
      resources: [{ type: "workspace", id: input.workspaceId }],
      context: { location: "", userAgent: undefined },
    });
  });
}
