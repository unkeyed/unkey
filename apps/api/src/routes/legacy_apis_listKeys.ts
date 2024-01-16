import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, isNull, sql } from "drizzle-orm";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/apis/{apiId}/keys",
  request: {
    header: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),
    params: z.object({
      apiId: z.string().min(1).openapi({
        description: "The id of the api to fetch",
        example: "api_1234",
      }),
    }),
    query: z.object({
      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of keys to return",
        example: 100,
      }),
      offset: z.coerce.number().optional().openapi({
        description:
          "Use this to fetch the next page of results. A new cursor will be returned in the response if there are more results.",
      }),
      ownerId: z.string().min(1).optional().openapi({
        description: "If provided, this will only return keys where the `ownerId` matches.",
      }),
    }),
  },
  responses: {
    200: {
      description: "Keys belonging to the api",
      content: {
        "application/json": {
          schema: z.object({
            keys: z.array(keySchema),
            total: z.number().int().openapi({
              description: "The total number of keys for this api",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type LegacyApisListKeysResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyApisListKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const auth = await rootKeyAuth(c);

    const apiId = c.req.param("apiId");
    const { limit, offset, ownerId } = c.req.query();

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `api ${apiId} not found`,
      });
    }

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${apiId} is not setup to handle keys`,
      });
    }
    const keysWhere: Parameters<typeof and> = [
      isNull(schema.keys.deletedAt),
      eq(schema.keys.keyAuthId, api.keyAuthId),
    ];
    if (ownerId) {
      keysWhere.push(eq(schema.keys.ownerId, ownerId));
    }

    const keys = await db.query.keys.findMany({
      where: and(...keysWhere),
      limit: parseInt(limit),
      orderBy: schema.keys.id,
      offset: offset ? parseInt(offset) : undefined,
    });

    const total = await db
      // @ts-ignore, mysql sucks
      .select({ count: sql<string>`count(*)` })
      .from(schema.keys)
      .where(and(eq(schema.keys.keyAuthId, api.keyAuthId), isNull(schema.keys.deletedAt)));

    return c.json({
      keys: keys.map((k) => ({
        id: k.id,
        start: k.start,
        apiId: api.id,
        workspaceId: k.workspaceId,
        name: k.name ?? undefined,
        ownerId: k.ownerId ?? undefined,
        meta: k.meta ? JSON.parse(k.meta) : undefined,
        createdAt: k.createdAt.getTime() ?? undefined,
        expires: k.expires?.getTime() ?? undefined,
        ratelimit:
          k.ratelimitType && k.ratelimitLimit && k.ratelimitRefillRate && k.ratelimitRefillInterval
            ? {
                type: k.ratelimitType,
                limit: k.ratelimitLimit,
                refillRate: k.ratelimitRefillRate,
                refillInterval: k.ratelimitRefillInterval,
              }
            : undefined,
        remaining: k.remaining ?? undefined,
      })),
      // @ts-ignore, mysql sucks
      total: parseInt(total.at(0)?.count ?? "0"),
      cursor: keys.at(-1)?.id ?? undefined,
    });
  });
