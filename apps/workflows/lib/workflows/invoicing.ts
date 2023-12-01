import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { inngest } from "@/lib/inngest";
import { Tinybird } from "@/lib/tinybird";
import Stripe from "stripe";


export const invoicing = inngest.createFunction(
  { id: "billing/invoicing" },
  { event: "billing.invoicing" },
  async ({ event, step, logger }) => {
    const db = connectDatabase();
    const stripe = new Stripe(env().STRIPE_SECRET_KEY, {
      apiVersion: "2022-11-15",
      typescript: true,
    });
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);

    const workspaces = await step.run("list workspaces", async () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, and }) => and(isNotNull(table.stripeCustomerId), isNotNull(table.subscriptions))
      }),
    );



    const now = new Date();
    const year = now.getUTCFullYear();
    const currentMonth = now.getUTCMonth() + 1; // months are 0 indexed


    await Promise.all(workspaces.map(async (workspace) => step.sendEvent("invoice.create", {
      workspace,
      year,
      month: currentMonth,
    })))



    return {
      event,
      body: summary,
    };
  },
);
