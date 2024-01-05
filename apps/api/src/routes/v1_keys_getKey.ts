import { cache, db, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/keys.getKey",
  request: {
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
export const registerV1KeysGetKey = (app: App) =>
  app.openapi(route, async (c) => {
    const authorization = c.req.header("authorization")?.replace("Bearer ", "");
    if (!authorization) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "key required",
      });
    }
    const rootKey = await keyService.verifyKey(c, { key: authorization });
    if (rootKey.error) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: rootKey.error.message,
      });
    }
    if (!rootKey.value.valid) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "the root key is not valid",
      });
    }
    if (!rootKey.value.isRootKey) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "root key required",
      });
    }

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

    if (!data || data.key.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }

    return c.json({
      id: data.key.id,
      start: data.key.start,
      apiId: data.api.id,
      workspaceId: data.key.workspaceId,
      name: data.key.name ?? undefined,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? JSON.parse(data.key.meta) : undefined,
      createdAt: data.key.createdAt.getTime(),
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remaining ?? undefined,
      refill:
        data.key.refillInterval && data.key.refillAmount
          ? {
              interval: data.key.refillInterval,
              amount: data.key.refillAmount,
              lastRefillAt: data.key.lastRefillAt?.getTime(),
            }
          : undefined,
      ratelimit:
        data.key.ratelimitType &&
        data.key.ratelimitLimit &&
        data.key.ratelimitRefillRate &&
        data.key.ratelimitRefillInterval
          ? {
              type: data.key.ratelimitType,
              limit: data.key.ratelimitLimit,
              refillRate: data.key.ratelimitRefillRate,
              refillInterval: data.key.ratelimitRefillInterval,
            }
          : undefined,
      enabled: data.key.enabled,
    });
  });
