import { client } from "@/trigger";

import { connectDatabase, eq, schema } from "@/lib/db";
import { eventTrigger } from "@trigger.dev/sdk";

client.defineJob({
  id: "billing.downgrade",
  name: "Downgrade workspaces when requested",
  version: "0.0.2",
  trigger: eventTrigger({
    name: "billing.downgrade",
  }),

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();

    const workspaces = await io.runTask("get workspace with downgrade request", async () =>
      db.query.workspaces.findMany({
        where: (table, { and, isNotNull }) => and(isNotNull(table.planDowngradeRequest)),
      }),
    );

    for (const ws of workspaces) {
      await io.runTask(`downgrade workspace ${ws.id}`, async () => {
        await db
          .update(schema.workspaces)
          .set({
            plan: ws.planDowngradeRequest,
            planChanged: null,
            planDowngradeRequest: null,
            subscriptions: ws.planDowngradeRequest === "free" ? null : undefined,
          })
          .where(eq(schema.workspaces.id, ws.id));
      });
    }
  },
});
