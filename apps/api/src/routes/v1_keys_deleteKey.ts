import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { schema } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";

const route = createRoute({
  method: "post",
  path: "/v1/keys.deleteKey",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().min(1).openapi({
              description: "The id of the key to revoke",
              example: "key_1234",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The key was successfully revoked, it may take up to 30s for this to take effect in all regions",
      content: {
        "application/json": {
          schema: z.object({}),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysDeleteKeyRequest = z.infer<
  typeof route.request.body.content["application/json"]["schema"]
>;
export type V1KeysDeleteKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysDeleteKey = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const { keyId } = c.req.valid("json");

    const data = await cache.withCache(c, "keyById", keyId, async () => {
      const dbRes = await db.query.keys.findFirst({
        where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAt)),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      });
      if (!dbRes) {
        return null;
      }
      return {
        key: dbRes,
        api: dbRes.keyAuth.api,
      };
    });

    if (!data || data.key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${keyId} not found` });
    }

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    await db.transaction(async (tx) => {
      await tx
        .update(schema.keys)
        .set({
          deletedAt: new Date(),
        })
        .where(eq(schema.keys.id, data.key.id));
      await tx.insert(schema.auditLogs).values({
        id: newId("auditLog"),
        time: new Date(),
        workspaceId: authorizedWorkspaceId,
        actorType: "key",
        actorId: rootKeyId,
        event: "key.delete",
        description: `revoked key ${data.key.id}`,
        keyAuthId: data.key.keyAuthId,
      });
    });

    await cache.remove(c, "keyById", data.key.id);
    await cache.remove(c, "keyByHash", data.key.hash);

    return c.json({});
  });
