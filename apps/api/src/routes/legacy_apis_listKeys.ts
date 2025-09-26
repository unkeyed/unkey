import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, isNull, sql } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { keySchema } from "./schema";

const route = createRoute({
  operationId: "deprecated.listKeys",
  summary: "List API keys (deprecated)",
  "x-speakeasy-ignore": true,
  "x-excluded": true,
  method: "get",
  path: "/v1/apis/{apiId}/keys",
  request: {
    header: z.object({
      authorization: z
        .string()
        .regex(/^Bearer [a-zA-Z0-9_]+/)
        .openapi({
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
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyApisListKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const { db, cache, metrics } = c.get("services");
    const auth = await rootKeyAuth(c);

    const apiId = c.req.param("apiId");
    const { limit, offset, ownerId } = c.req.valid("query");

    const { val: api, err } = await cache.apiById.swr(apiId, async () => {
      return (
        (await db.readonly.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAtM)),
          with: {
            keyAuth: true,
          },
        })) ?? null
      );
    });

    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load api: ${err.message}`,
      });
    }
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
      isNull(schema.keys.deletedAtM),
      eq(schema.keys.keyAuthId, api.keyAuthId),
    ];
    if (ownerId) {
      keysWhere.push(eq(schema.keys.ownerId, ownerId));
    }

    const dbStart = performance.now();
    const keys = await db.readonly.query.keys.findMany({
      where: and(...keysWhere),
      limit: limit,
      orderBy: schema.keys.id,
      offset: offset ? offset : undefined,
      with: {
        ratelimits: true,
      },
    });
    metrics.emit({
      metric: "metric.db.read",
      query: "getKeysByKeyAuthId",
      latency: performance.now() - dbStart,
    });

    const total = await db.readonly
      .select({ count: sql<string>`count(*)` })
      .from(schema.keys)
      .where(and(eq(schema.keys.keyAuthId, api.keyAuthId), isNull(schema.keys.deletedAtM)));

    return c.json({
      keys: keys.map((k) => {
        const ratelimit = k.ratelimits.find((rl) => rl.name === "default");
        return {
          id: k.id,
          start: k.start,
          apiId: api.id,
          workspaceId: k.workspaceId,
          name: k.name ?? undefined,
          ownerId: k.ownerId ?? undefined,
          meta: k.meta ? JSON.parse(k.meta) : undefined,
          createdAt: k.createdAtM ?? undefined,
          expires: k.expires?.getTime() ?? undefined,
          ratelimit: ratelimit
            ? {
                async: false,
                limit: ratelimit.limit,
                duration: ratelimit.duration,
                refillRate: ratelimit.limit,
                refillInterval: ratelimit.duration,
              }
            : undefined,
          remaining: k.remaining ?? undefined,
        };
      }),
      // @ts-ignore, mysql sucks
      total: Number.parseInt(total.at(0)?.count ?? "0"),
      cursor: keys.at(-1)?.id ?? undefined,
    });
  });
