import { logger, schedules } from "@trigger.dev/sdk/v3";

import { connectDatabase } from "@/lib/db";

import { createInvoiceTask } from "./create-invoice";
import { downgradeTask } from "./downgrade-requests";

export const invoicingTask = schedules.task({
  id: "billing_invoicing",
  run: async () => {
    logger.info("task starting..");
    const db = connectDatabase();

    let workspaces = await db.query.workspaces.findMany({
      where: (table, { isNotNull, isNull, not, eq, and }) =>
        and(
          isNotNull(table.stripeCustomerId),
          isNotNull(table.subscriptions),
          not(eq(table.plan, "free")),
          isNull(table.deletedAt),
        ),
    });
    // hack to filter out workspaces with `{}` as subscriptions
    workspaces = workspaces.filter(
      (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
    );

    logger.info(`found ${workspaces.length} workspaces`);

    /**
     * Dates gymnastics to get the previous month's number, ie: if it's December now, it returns -> 11
     */
    const t = new Date();
    t.setUTCMonth(t.getUTCMonth() - 1);
    const year = t.getUTCFullYear();
    const month = t.getUTCMonth() + 1; // months are 0 indexed

    if (workspaces.length > 0) {
      await createInvoiceTask.batchTriggerAndWait(
        workspaces.map((ws) => ({
          payload: {
            workspaceId: ws.id,
            year,
            month,
          },
        })),
      );
    }

    await downgradeTask.triggerAndWait();

    return {
      workspaceIds: workspaces.map((w) => w.id),
    };
  },
});
