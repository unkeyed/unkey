import { connectDatabase } from "@/lib/db";
import { inngest } from "@/lib/inngest";

export const invoicing = inngest.createFunction(
  {
    id: "billing/invoicing",
  },
  { cron: "0 12 1 * *" }, // every 1st of the month at noon UTC
  async ({ event, step, logger }) => {
    const db = connectDatabase();

    let workspaces = await step.run("list workspaces", async () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, not, eq, and }) =>
          and(
            isNotNull(table.stripeCustomerId),
            isNotNull(table.subscriptions),
            not(eq(table.plan, "free")),
          ),
      }),
    );
    // hack to filter out workspaces with `{}` as subscriptions
    workspaces = workspaces.filter(
      (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
    );

    logger.info(`found ${workspaces.length} workspaces`, JSON.stringify(workspaces));

    /**
     * Dates gymnastics to get the previous month's number, ie: if it's December now, it returns -> 11
     */
    const t = new Date();
    t.setUTCMonth(t.getUTCMonth() - 1);
    const year = t.getUTCFullYear();
    const month = t.getUTCMonth() + 1; // months are 0 indexed

    await Promise.all(
      workspaces.map(async (workspace) =>
        step.sendEvent("invoice.create", {
          name: "billing/create.invoice",
          data: {
            workspaceId: workspace.id,
            year,
            month,
          },
        }),
      ),
    );

    return {
      event,
      body: "done",
    };
  },
);
