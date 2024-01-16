import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
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
export type LegacyKeysGetKeyResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyKeysGet = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const { keyId } = c.req.param();

    if (!keyId) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "no key id given",
      });
    }

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
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }

    return c.json({
      id: data.key.id,
      apiId: data.api.id,
      workspaceId: data.key.workspaceId,
      name: data.key.name ?? undefined,
      start: data.key.start,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? JSON.parse(data.key.meta) : undefined,
      createdAt: data.key.createdAt?.getTime() ?? undefined,
      forWorkspaceId: data.key.forWorkspaceId ?? undefined,
      expiresAt: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remaining ?? undefined,
      rateLimit: data.key.ratelimitType
        ? {
            type: data.key.ratelimitType ?? undefined,
            limit: data.key.ratelimitLimit ?? undefined,
            refillRate: data.key.ratelimitRefillRate ?? undefined,
            refillInternal: data.key.ratelimitRefillInterval ?? undefined,
          }
        : undefined,
    });
  });
