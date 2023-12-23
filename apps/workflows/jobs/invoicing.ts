import { connectDatabase } from "@/lib/db";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";

client.defineJob({
  id: "billing.invoicing",
  name: "Monthly invoicing",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 12 1 * *", // every 1st of the month at noon UTC
  }),
  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();

    let workspaces = await io.runTask("list workspaces", async () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, isNull, not, eq, and }) =>
          and(
            isNotNull(table.stripeCustomerId),
            isNotNull(table.subscriptions),
            not(eq(table.plan, "free")),
            isNull(table.deletedAt),
          ),
      }),
    );
    // hack to filter out workspaces with `{}` as subscriptions
    workspaces = workspaces.filter(
      (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
    );

    io.logger.info(`found ${workspaces.length} workspaces`, workspaces);

    /**
     * Dates gymnastics to get the previous month's number, ie: if it's December now, it returns -> 11
     */
    const t = new Date();
    t.setUTCMonth(t.getUTCMonth() - 1);
    const year = t.getUTCFullYear();
    const month = t.getUTCMonth() + 1; // months are 0 indexed

    await io.sendEvents(
      "delegate invoicing",
      workspaces.map((w) => ({
        name: "billing.invoicing.createInvoice",
        payload: {
          workspaceId: w.id,
          year,
          month,
        },
      })),
    );
    return {
      workspaceIds: workspaces.map((w) => w.id),
    };
  },
});
