import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import type { CacheNamespaces } from "@/pkg/cache/namespaces";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { buildUnkeyQuery, type unkeyPermissionValidation } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "keys.getVerifications",
  method: "get",
  path: "/v1/keys.getVerifications",
  "x-speakeasy-name-override": "getVerifications",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
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
          where: (table, { eq, and, isNull }) => and(eq(table.id, keyId), isNull(table.deletedAtM)),
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
            ratelimits: true,
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
          ratelimits: dbRes.ratelimits,
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
            and(eq(table.ownerId, ownerId), isNull(table.deletedAtM)),
          with: {
            encrypted: true,
            keyAuth: {
              with: {
                api: true,
              },
            },
            ratelimits: true,
          },
        });
        if (!dbRes) {
          return [];
        }
        return dbRes.map((key) => ({
          key,
          api: key.keyAuth.api,
          ratelimits: key.ratelimits,
        }));
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
          const res = await analytics
            .getVerificationsDaily({
              workspaceId: authorizedWorkspaceId,
              keyspaceId: keySpaceId,
              keyId: keyId,
              startTime: start ? start : now - 24 * 60 * 60 * 1000,
              endTime: end ? end : now,
              identities: null,
              keyIds: null,
              outcomes: null,
              names: null,
              tags: null,
            })
            .catch((err) => {
              throw new Error(err.message);
            });

          return transformData(res);
        });
      }),
    );

    const verifications: {
      [time: number]: {
        success: number;
        rateLimited: number;
        usageExceeded: number;
      };
    } = {};
    for (const dataPoint of verificationsFromAllKeys) {
      if (dataPoint.err) {
        logger.error(dataPoint.err.message);
        continue;
      }
      for (const d of dataPoint.val!) {
        if (!verifications[d.time]) {
          verifications[d.time] = {
            success: 0,
            rateLimited: 0,
            usageExceeded: 0,
          };
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

function transformData(
  data: VerificationTimeseriesDataPoint[] | undefined,
): CacheNamespaces["verificationsByKeyId"] {
  if (!data || !data.length) {
    return [];
  }

  const verificationsByKeyId = data.flatMap((item) => {
    const time = item.x;
    const outcomes: Array<{ outcome: string; count: number }> = [
      { outcome: "valid", count: item.y.valid_count },
      { outcome: "rate_limited", count: item.y.rate_limited_count },
      {
        outcome: "insufficient_permissions",
        count: item.y.insufficient_permissions_count,
      },
      { outcome: "forbidden", count: item.y.forbidden_count },
      { outcome: "disabled", count: item.y.disabled_count },
      { outcome: "expired", count: item.y.expired_count },
      { outcome: "usage_exceeded", count: item.y.usage_exceeded_count },
    ];

    // Only include outcomes with non-zero counts
    return outcomes
      .filter((outcome) => outcome.count > 0)
      .map((outcome) => ({
        time,
        count: outcome.count,
        outcome: outcome.outcome,
      }));
  });

  return verificationsByKeyId;
}
