import { z } from "zod";

import { client } from "@/trigger";

import { connectDatabase, eq, schema } from "@/lib/db";
import { eventTrigger } from "@trigger.dev/sdk";
import { newId } from "@unkey/id";

client.defineJob({
  id: "resources.keys.deleteKey",
  name: "Soft delete a key",
  version: "0.0.1",
  trigger: eventTrigger({
    name: "resources.keys.deleteKey",
    schema: z.object({
      keyId: z.string(),
      apiId: z.string(),
      workspaceId: z.string(),
      actor: z.object({
        type: z.enum(["user", "key"]),
        id: z.string(),
      }),
    }),
  }),

  run: async (payload, io, _ctx) => {
    const { workspaceId, apiId, keyId, actor } = payload;

    const db = connectDatabase();

    await io.runTask(`soft delete key ${keyId}`, async () => {
      await db.update(schema.keys).set({ deletedAt: new Date() }).where(eq(schema.keys.id, keyId));
    });

    await io.runTask("create audit log", async () => {
      const auditLogId = newId("auditLog");
      await db.insert(schema.auditLogs).values({
        id: auditLogId,
        workspaceId,
        apiId,
        keyId,
        event: "key.delete",
        description: `Key ${keyId} deleted`,
        time: new Date(),
        actorType: actor.type,
        actorId: actor.id,
      });
      return { auditLogId };
    });
  },
});
