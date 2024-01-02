import { analytics, cache, db, keyService } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";

const route = createRoute({
  method: "get",
  path: "/vx/keys.getVerifications",
  request: {
    query: z.object({
      keyId: z.string().min(1).openapi({
        description: "The id of the key to fetch",
        example: "key_1234",
      }),
      start: z.coerce.number().int().optional().openapi({
        description: "The start of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      end: z.coerce.number().int().optional().openapi({
        description: "The end of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      granularity: z.enum(["hour", "day", "month"]).optional().default("day").openapi({
        description: "The granularity of the usage data to fetch",
        example: "day",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for a single key",
      content: {
        "application/json": {
          schema: z.object({
            verifications: z.array(
              z.object({
                time: z.number().int().positive().openapi({
                  description: "The timestamp of the usage data",
                  example: 1620000000000,
                }),
                success: z.number().openapi({
                  description: "The number of successful requests",
                  example: 100,
                }),
                rateLimited: z.number().openapi({
                  description: "The number of requests that were rate limited",
                  example: 10,
                }),
                usageExceeded: z.number().openapi({
                  description: "The number of requests that exceeded the usage limit",
                  example: 0,
                }),
              }),
            ),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysGetVerificationsResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;
export const registerV1KeysGetVerifications = (app: App) =>
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

    const { keyId, start, end } = c.req.query();

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

    const verifications = await analytics.getVerificationsDaily({
      workspaceId: data.key.workspaceId,
      apiId: data.api.id,
      keyId: data.key.id,
      start: start ? parseInt(start) : undefined,
      end: end ? parseInt(end) : undefined,
    });

    return c.json({
      verifications: verifications.data,
    });
  });
