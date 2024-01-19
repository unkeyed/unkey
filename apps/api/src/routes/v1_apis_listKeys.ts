import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, gt, isNull, sql } from "drizzle-orm";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { buildQuery } from "@unkey/rbac";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/apis.listKeys",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      apiId: z.string().min(1).openapi({
        description: "The id of the api to fetch",
        example: "api_1234",
      }),
      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of keys to return",
        example: 100,
      }),
      cursor: z.string().optional().openapi({
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
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            keys: z.array(keySchema),
            cursor: z.string().optional().openapi({
              description:
                "The cursor to use for the next page of results, if no cursor is returned, there are no more results",
              example: "eyJrZXkiOiJrZXlfMTIzNCJ9",
            }),
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
export type V1ApisListKeysResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1ApisListKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId, limit, cursor, ownerId } = c.req.query();

    const auth = await rootKeyAuth(
      c,
      buildQuery(({ or }) => or("*", `api.${apiId}.read_api`)),
    );

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

    const keys = await db.query.keys.findMany({
      where: and(
        ...[
          eq(schema.keys.keyAuthId, api.keyAuthId),
          isNull(schema.keys.deletedAt),
          cursor ? gt(schema.keys.id, cursor) : undefined,
          ownerId ? eq(schema.keys.ownerId, ownerId) : undefined,
        ].filter(Boolean),
      ),
      limit: parseInt(limit),
      orderBy: schema.keys.id,
    });

    const total = await db
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
        refill:
          k.refillInterval && k.refillAmount && k.lastRefillAt
            ? {
                interval: k.refillInterval,
                amount: k.refillAmount,
                lastRefillAt: k.lastRefillAt?.getTime(),
              }
            : undefined,
      })),
      // @ts-ignore, mysql sucks
      total: parseInt(total.at(0)?.count ?? "0"),
      cursor: keys.at(-1)?.id ?? undefined,
    });
  });
