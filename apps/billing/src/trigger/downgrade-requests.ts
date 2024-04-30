import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { eventTrigger } from "@trigger.dev/sdk";
import { logger, task } from "@trigger.dev/sdk/v3";

export const downgradeTask = task({
  id: "billing_downgrade",

  run: async () => {
    const db = connectDatabase();

    const workspaces = await db.query.workspaces.findMany({
      where: (table, { and, isNotNull }) => and(isNotNull(table.planDowngradeRequest)),
    });

    for (const ws of workspaces) {
      await await db
        .update(schema.workspaces)
        .set({
          plan: ws.planDowngradeRequest,
          planChanged: null,
          planDowngradeRequest: null,
          subscriptions: ws.planDowngradeRequest === "free" ? null : undefined,
        })
        .where(eq(schema.workspaces.id, ws.id));
    }
  },
});
