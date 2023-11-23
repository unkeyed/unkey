import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { rootKey } from "@/pkg/hono/middlewares/root-key-middleware";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/keys.getKey",
  request: {
    header: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),
    query: z.object({
      keyId: z.string().min(1).openapi({
        description: "The id of the key to fetch",
        example: "key_1234",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema: keySchema,
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysGetKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;
export const registerV1KeysGetKey = (app: App) => {
  app.use(route.getRoutingPath(), rootKey());
  return app.openapi(route, async (c) => {
    const rootKey = c.get("rootKey");

    const { keyId } = c.req.query();

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

    if (!data || data.key.workspaceId !== rootKey.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${keyId} not found` });
    }

    return c.jsonT({
      id: data.key.id,
      start: data.key.start,
      apiId: data.api.id,
      workspaceId: data.key.workspaceId,
      name: data.key.name ?? undefined,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ?? undefined,
      createdAt: data.key.createdAt.getTime() ?? undefined,
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remaining ?? undefined,
    });
  });
};
