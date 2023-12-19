import { client } from "@/trigger";
import { z } from "zod";

import { connectDatabase, eq, schema } from "@/lib/db";
import { eventTrigger } from "@trigger.dev/sdk";
import { newId } from "@unkey/id";

client.defineJob({
  id: "resources.apis.deleteApi",
  name: "Soft delete an api and all its keys",
  version: "0.0.1",
  trigger: eventTrigger({
    name: "resources.apis.deleteApi",
    schema: z.object({
      workspaceId: z.string(),
      apiId: z.string(),
      actor: z.object({
        type: z.enum(["user", "key"]),
        id: z.string(),
      }),
    }),
  }),
  run: async ({ workspaceId, apiId, actor }, io, _ctx) => {
    const db = connectDatabase();

    const api = await io.runTask(`get api ${apiId}`, async () => {
      return db.query.apis.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
      });
    });

    if (!api) {
      throw new Error(`api ${apiId} not found`);
    }

    const keyAuthId = api.keyAuthId;
    if (!keyAuthId) {
      return;
    }
    const keys = await io.runTask(`get keys for api ${apiId}`, async () => {
      return db.query.keys.findMany({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.keyAuthId, keyAuthId), isNull(table.deletedAt)),
        columns: {
          id: true,
        },
      });
    });

    for (const key of keys) {
      await io.runTask(`soft delete key ${key.id}`, async () => {
        await db
          .update(schema.keys)
          .set({ deletedAt: new Date() })
          .where(eq(schema.keys.id, key.id));
      });

      await io.runTask(`create audit log for ${key.id}`, async () => {
        const auditLogId = newId("auditLog");
        await db.insert(schema.auditLogs).values({
          id: auditLogId,
          workspaceId,
          apiId,
          keyId: key.id,
          event: "key.delete",
          description: `Key ${key.id} deleted`,
          time: new Date(),
          actorType: actor.type,
          actorId: actor.id,
        });
        return { auditLogId };
      });
    }

    await io.runTask(`soft delete api ${apiId}`, async () => {
      await db
        .update(schema.apis)
        .set({ deletedAt: new Date(), state: null })
        .where(eq(schema.apis.id, api.id));
    });
    await io.runTask("create audit log for api", async () => {
      const auditLogId = newId("auditLog");
      await db.insert(schema.auditLogs).values({
        id: auditLogId,
        workspaceId,
        apiId: apiId,
        event: "api.delete",
        description: `Api ${apiId} deleted`,
        time: new Date(),
        actorType: actor.type,
        actorId: actor.id,
      });
      return { auditLogId };
    });
  },
});
