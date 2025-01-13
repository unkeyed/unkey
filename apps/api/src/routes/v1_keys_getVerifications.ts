import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery, type unkeyPermissionValidation } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "keys.getVerifications",
  method: "get",
  path: "/v1/keys.getVerifications",
  "x-speakeasy-name-override": "getVerifications",
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
        description:
          "The start of the period to fetch usage for as unix milliseconds timestamp, defaults to 24h ago.",
        example: 1620000000000,
      }),
      end: z.coerce.number().int().optional().openapi({
        description:
          "The end of the period to fetch usage for as unix milliseconds timestamp, defaults to now.",
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
      description: "Usage numbers over time",
      content: {
        "application/json": {
          schema: z.object({
            verifications: z.array(
              z.object({
                time: z.number().int().positive().openapi({
                  description: "The timestamp of the usage data",
                  example: 1620000000000,
                }),
                success: z.number().int().openapi({
                  description: "The number of successful requests",
                  example: 100,
                }),
                rateLimited: z.number().int().openapi({
                  description: "The number of requests that were rate limited",
                  example: 10,
                }),
                usageExceeded: z.number().int().openapi({
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
    const { keyId, ownerId, start, end } = c.req.valid("query");

    const { analytics, cache, db, logger } = c.get("services");

    const ids: {
      keyId: string;
      apiId: string;
      keySpaceId: string;
      workspaceId: string;
    }[] = [];

    if (keyId) {
      const data = await cache.keyById.swr(keyId, async (keyId) => {
        const dbRes = await db.readonly.query.keys.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAt)),
          with: {
            identity: true,
            encrypted: true,
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
          identity: dbRes.identity
            ? {
                id: dbRes.identity.id,
                externalId: dbRes.identity.externalId,
                meta: dbRes.identity.meta,
              }
            : null,
        };
      });

      if (data.err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: `Unable to load key by id: ${data.err.message}`,
        });
      }
      if (!data.val) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `key ${keyId} not found`,
        });
      }

      ids.push({
        keyId,
        apiId: data.val.api.id,
        keySpaceId: data.val.api.keyAuthId!,
        workspaceId: data.val.key.workspaceId,
      });
    } else {
      if (!ownerId) {
        throw new UnkeyApiError({
          code: "BAD_REQUEST",
          message: "keyId or ownerId must be provided",
        });
      }

      const keys = await cache.keysByOwnerId.swr(ownerId, async () => {
        const dbRes = await db.readonly.query.keys.findMany({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.ownerId, ownerId), isNull(table.deletedAt)),
          with: {
            encrypted: true,
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
      if (keys.err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: `Unbable to load keys by ownerId: ${keys.err.message}`,
        });
      }

      ids.push(
        ...(keys.val ?? []).map(({ key, api }) => ({
          keyId: key.id,
          apiId: api.id,
          keySpaceId: api.keyAuthId!,
          workspaceId: key.workspaceId,
        })),
      );
    }

    if (ids.length === 0) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "No keys were found to match your query",
      });
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
    const now = Date.now();

    const verificationsFromAllKeys = await Promise.all(
      ids.map(({ keyId, keySpaceId }) => {
        return cache.verificationsByKeyId.swr(`${keyId}:${start}-${end}`, async () => {
          const res = await analytics.getVerificationsDaily({
            workspaceId: authorizedWorkspaceId,
            keySpaceId: keySpaceId,
            keyId: keyId,
            start: start ? start : now - 24 * 60 * 60 * 1000,
            end: end ? end : now,
          });
          if (res.err) {
            throw new Error(res.err.message);
          }
          return res.val;
        });
      }),
    );

    const verifications: {
      [time: number]: { success: number; rateLimited: number; usageExceeded: number };
    } = {};
    for (const dataPoint of verificationsFromAllKeys) {
      if (dataPoint.err) {
        logger.error(dataPoint.err.message);
        continue;
      }
      for (const d of dataPoint.val!) {
        if (!verifications[d.time]) {
          verifications[d.time] = { success: 0, rateLimited: 0, usageExceeded: 0 };
        }
        switch (d.outcome) {
          case "VALID":
            verifications[d.time].success += d.count;
            break;
          case "RATE_LIMITED":
            verifications[d.time].rateLimited += d.count;
            break;
          case "USAGE_EXCEEDED":
            verifications[d.time].usageExceeded += d.count;
            break;
        }
      }
    }

    // really ugly hack to return an emoty array in case there wasn't a single verification
    // this became necessary when we switched to clickhouse, due to the different responses
    if (
      Object.values(verifications).every(({ success, rateLimited, usageExceeded }) => {
        return success + rateLimited + usageExceeded === 0;
      })
    ) {
      return c.json({ verifications: [] });
    }
    return c.json({
      verifications: Object.entries(verifications).map(
        ([time, { success, rateLimited, usageExceeded }]) => ({
          time: Number.parseInt(time),
          success,
          rateLimited,
          usageExceeded,
        }),
      ),
    });
  });
