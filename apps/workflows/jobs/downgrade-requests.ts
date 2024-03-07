import { client } from "@/trigger";

import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
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
    const tb = new Tinybird(env().TINYBIRD_TOKEN);

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
      await io.runTask(`create audit log for downgrading ${ws.id}`, async () => {
        await tb.ingestAuditLogs({
          workspaceId: ws.id,
          event: "workspace.update",
          actor: {
            type: "system",
            id: "trigger",
          },
          description: `Downgraded ${ws.id} to ${ws.planDowngradeRequest}`,
          resources: [
            {
              type: "workspace",
              id: ws.id,
            },
          ],
          context: {
            location: "trigger",
          },
        });
      });
    }
  },
});
