import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/keys.getKey",
  security: [{ bearerAuth: [] }],
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
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1KeysGetKey = (app: App) =>
  app.openapi(route, async (c) => {
    const { keyId } = c.req.query();
    const { cache, db } = c.get("services");

    const { val: data, err } = await cache.withCache(c, "keyById", keyId, async () => {
      const dbRes = await db.readonly.query.keys.findFirst({
        where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAt)),
        with: {
          permissions: { with: { permission: true } },
          roles: { with: { role: true } },
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
        permissions: dbRes.permissions.map((p) => p.permission.name),
        roles: dbRes.roles.map((p) => p.role.name),
      };
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load key: ${err.message}`,
      });
    }
    if (!data) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }
    const { api, key } = data;
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.read_key", `api.${api.id}.read_key`)),
    );

    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${keyId} not found`,
      });
    }
    let meta = key.meta ? JSON.parse(key.meta) : undefined;
    if (!meta || Object.keys(meta).length === 0) {
      meta = undefined;
    }
    return c.json({
      id: key.id,
      start: key.start,
      apiId: api.id,
      workspaceId: key.workspaceId,
      name: key.name ?? undefined,
      ownerId: key.ownerId ?? undefined,
      meta: key.meta ? JSON.parse(key.meta) : undefined,
      createdAt: key.createdAt.getTime(),
      expires: key.expires?.getTime() ?? undefined,
      remaining: key.remaining ?? undefined,
      refill:
        key.refillInterval && key.refillAmount
          ? {
              interval: key.refillInterval,
              amount: key.refillAmount,
              lastRefillAt: key.lastRefillAt?.getTime(),
            }
          : undefined,
      ratelimit:
        key.ratelimitType &&
        key.ratelimitLimit &&
        key.ratelimitRefillRate &&
        key.ratelimitRefillInterval
          ? {
              type: key.ratelimitType,
              limit: key.ratelimitLimit,
              refillRate: key.ratelimitRefillRate,
              refillInterval: key.ratelimitRefillInterval,
            }
          : undefined,
      roles: data.roles,
      permissions: data.permissions,
      enabled: key.enabled,
    });
  });
