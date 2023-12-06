import { cache, db, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, gt, isNull, sql } from "drizzle-orm";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { keySchema } from "./schema";

const route = createRoute({
  method: "get",
  path: "/v1/apis.listKeys",
  request: {
    query: z.object({
      apiId: z.string().min(1).openapi({
        description: "The id of the api to fetch",
        example: "api_1234",
      }),
      limit: z.coerce.number().int().min(1).max(100).default(100).openapi({
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
    const authorization = c.req.header("authorization")?.replace("Bearer ", "");
    if (!authorization) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "key required" });
    }
    const rootKey = await keyService.verifyKey(c, { key: authorization });
    if (rootKey.error) {
      throw new UnkeyApiError({ code: "INTERNAL_SERVER_ERROR", message: rootKey.error.message });
    }
    if (!rootKey.value.valid) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "the root key is not valid" });
    }
    if (!rootKey.value.isRootKey) {
      throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "root key required" });
    }

    const { apiId, limit, cursor, ownerId } = c.req.query();

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq }) => eq(table.id, apiId),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
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
    if (cursor) {
      keysWhere.push(gt(schema.keys.id, cursor));
    }
    if (ownerId) {
      keysWhere.push(eq(schema.keys.ownerId, ownerId));
    }

    const [keys, total] = await Promise.all([
      db.query.keys.findMany({
        where: and(...keysWhere),
        limit: parseInt(limit),
        orderBy: schema.keys.id,
      }),
      db
        .select({ count: sql<string>`count(*)` })
        .from(schema.keys)
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId), isNull(schema.keys.deletedAt))),
    ]);

    if (!api || api.workspaceId !== rootKey.value.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }

    return c.json({
      keys: keys.map((k) => ({
        id: k.id,
        ownerId: k.ownerId,
        createdAt: k.createdAt,
      })),
      total: parseInt(total.at(0)?.count ?? "0"),
      cursor: keys.at(-1)?.id ?? undefined,
    });
  });
