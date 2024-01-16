import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { schema } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { newId } from "@unkey/id";
import { eq } from "drizzle-orm";

const route = createRoute({
  method: "delete",
  path: "/v1/keys/:keyId",
  request: {
    headers: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),
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
export type LegacyKeysDeleteKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyKeysDelete = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const { keyId } = c.req.param();

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
        actorId: auth.key!.id,
        event: "key.delete",
        description: `Key ${data.key.id} deleted`,
        apiId: data.api.id,
      });
    });
    await cache.remove(c, "keyById", data.key.id);
    await cache.remove(c, "keyByHash", data.key.hash);

    return c.json({});
  });
