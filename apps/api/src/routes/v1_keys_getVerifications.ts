import { analytics, cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery, unkeyPermissionValidation } from "@unkey/rbac";

const route = createRoute({
  method: "get",
  path: "/v1/keys.getVerifications",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      keyId: z.string().optional().openapi({
        description: "The id of the key to fetch, either `keyId` or `ownerId` must be provided",
        example: "key_1234",
      }),
      ownerId: z.string().optional().openapi({
        description: "The owner id to fetch keys for, either `keyId` or `ownerId` must be provided",
        example: "chronark",
      }),
      start: z.coerce.number().int().optional().openapi({
        description: "The start of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      end: z.coerce.number().int().optional().openapi({
        description: "The end of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      granularity: z.enum(["day"]).optional().default("day").openapi({
        description:
          "The granularity of the usage data to fetch, currently only `day` is supported",
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
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1KeysGetVerifications = (app: App) =>
  app.openapi(route, async (c) => {
    const { keyId, ownerId, start, end } = c.req.query();

    const ids: {
      keyId: string;
      apiId: string;
      workspaceId: string;
    }[] = [];

    if (keyId) {
      const data = await cache.withCache(c, "keyById", keyId, async () => {
        const dbRes = await db.query.keys.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAt)),
          with: {
            permissions: { with: { permission: true } },
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
          permissions: dbRes.permissions.map((p) => p.permission.key!).filter(Boolean),
        };
      });

      if (!data) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `key ${keyId} not found`,
        });
      }
      ids.push({ keyId, apiId: data.api.id, workspaceId: data.key.workspaceId });
    } else {
      if (!ownerId) {
        throw new UnkeyApiError({
          code: "BAD_REQUEST",
          message: "keyId or ownerId must be provided",
        });
      }

      const keys = await cache.withCache(c, "keysByOwnerId", ownerId, async () => {
        const dbRes = await db.query.keys.findMany({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.ownerId, ownerId), isNull(table.deletedAt)),
          with: {
            keyAuth: {
              with: {
                api: true,
              },
            },
          },
        });
        if (!dbRes) {
          return [];
        }
        return dbRes.map((key) => ({ key, api: key.keyAuth.api }));
      });

      ids.push(
        ...keys.map(({ key, api }) => ({
          keyId: key.id,
          apiId: api.id,
          workspaceId: key.workspaceId,
        })),
      );
    }

    const apiIds = Array.from(new Set(ids.map(({ apiId }) => apiId)));
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or, and }) =>
        or(
          "*",
          "api.*.read_key",
          and(
            ...apiIds.map(
              (apiId) =>
                `api.${apiId}.read_key` satisfies z.infer<typeof unkeyPermissionValidation>,
            ),
          ),
        ),
      ),
    );
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    if (ids.some(({ workspaceId }) => workspaceId !== authorizedWorkspaceId)) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "you are not allowed to access this workspace",
      });
    }

    const verificationsFromAllKeys = await Promise.all(
      ids.map(({ keyId, apiId }) => {
        return cache.withCache(c, "verificationsByKeyId", `${keyId}:${start}-${end}`, async () => {
          const res = await analytics.getVerificationsDaily({
            workspaceId: authorizedWorkspaceId,
            apiId: apiId,
            keyId: keyId,
            start: start ? parseInt(start) : undefined,
            end: end ? parseInt(end) : undefined,
          });
          return res.data;
        });
      }),
    );

    const verifications: {
      [time: number]: { success: number; rateLimited: number; usageExceeded: number };
    } = {};
    for (const dataPoint of verificationsFromAllKeys) {
      for (const d of dataPoint) {
        if (!verifications[d.time]) {
          verifications[d.time] = { success: 0, rateLimited: 0, usageExceeded: 0 };
        }
        verifications[d.time].success += d.success;
        verifications[d.time].rateLimited += d.rateLimited;
        verifications[d.time].usageExceeded += d.usageExceeded;
      }
    }

    return c.json({
      verifications: Object.entries(verifications).map(
        ([time, { success, rateLimited, usageExceeded }]) => ({
          time: parseInt(time),
          success,
          rateLimited,
          usageExceeded,
        }),
      ),
    });
  });
