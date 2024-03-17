import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { Analytics } from "@/pkg/analytics";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { Api, Key } from "@unkey/db";
import { buildUnkeyQuery, unkeyPermissionValidation } from "@unkey/rbac";

const route = createRoute({
  method: "get",
  path: "/v1/analytics.getByOwnerId",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      ownerId: z.string().openapi({
        description: "The owner id to fetch keys for, `ownerId` must be provided",
        example: "chronark",
      }),
      apiId: z.string().optional().openapi({
        description: "The id of the api to fetch keys for",
        example: "api_1234",
      }),
      start: z.coerce.number().int().optional().openapi({
        description: "The start of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      end: z.coerce.number().int().optional().openapi({
        description: "The end of the period to fetch usage for as unix milliseconds timestamp",
        example: 1620000000000,
      }),
      granularity: z
        .enum(["hourly", "daily", "weekly", "monthly"])
        .optional()
        .default("daily")
        .openapi({
          description: "The granularity of the usage data to fetch, default value is 'daily'.",
          example: "daily",
        }),
    }),
  },
  responses: {
    200: {
      description: "Usage numbers over time",
      content: {
        "application/json": {
          schema: z.object({
            ownerId: z.string().openapi({
              description:
                "The owner id used to fetch keys. This is the same as the `ownerId` provided",
              example: "chronark",
            }),
            apis: z.array(
              z.object({
                apiId: z.string().openapi({
                  description: "The id of the api",
                  example: "api_1234",
                }),
                apiName: z.string().openapi({
                  description: "The name of the api",
                  example: "my-api",
                }),
                keys: z.array(
                  z.string().openapi({
                    description: "The keys of the api",
                    example: "key_1234",
                  }),
                ),
              }),
            ),
            verificationsByDate: z.array(
              z.object({
                time: z.string().openapi({
                  description: "The timestamp of the usage data",
                  example: "2024-01-01",
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
export const registerV1AnalyticsGetByOwnerId = (app: App) =>
  app.openapi(route, async (c) => {
    const { ownerId, apiId, start, end, granularity } = c.req.query();
    const { analytics, cache, db } = c.get("services");
    function getVerificationsByOwnerId(granularity: string, analytics: Analytics) {
      switch (granularity) {
        case "hourly":
          return analytics.getVerificationsByOwnerIdHourly;
        case "daily":
          return analytics.getVerificationsByOwnerIdDaily;
        case "weekly":
          return analytics.getVerificationsByOwnerIdWeekly;
        case "monthly":
          return analytics.getVerificationsByOwnerIdMonthly;
        default:
          return analytics.getVerificationsByOwnerIdDaily;
      }
    }
    const getVerifications = getVerificationsByOwnerId(granularity, analytics);
    const keysList: {
      key: Key;
      api: Api;
      permissions: string[];
      roles: string[];
    }[] = [];

    //Get all keys by ownerId and optionally apiId
    const keys = await cache.withCache(c, "analyticsByOwnerId", ownerId, async () => {
      const dbRes = await db.query.keys.findMany({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.ownerId, ownerId), isNull(table.deletedAt)),
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
        return [];
      }
      return dbRes.map((val) => {
        return {
          key: val,
          api: val.keyAuth.api,
          permissions: val.permissions.map((val) => val.permission.name),
          roles: val.roles.map((val) => val.role.name),
        };
      });
    });
    if (keys.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Unbable to load keys by ownerId: ${keys.err.message}`,
      });
    }
    if (!keys.val) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "No keys were found to match your query",
      });
    }

    keys.val.map((val) => {
      keysList.push({
        key: val.key,
        api: val.api,
        permissions: val.permissions,
        roles: val.roles,
      });
    });

    if (keysList.length === 0) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "No keys were found to match your query",
      });
    }

    const apiIds = Array.from(new Set(keysList.map(({ api }) => api.id)));

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
    const authorizedKeys = keysList.filter(({ api }) => api.workspaceId === authorizedWorkspaceId);

    if (authorizedKeys.length === 0) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "you are not allowed to access this workspace",
      });
    }

    const verificationsFromAllKeys = await getVerifications({
      workspaceId: authorizedWorkspaceId,
      ownerId: ownerId,
      apiId: apiId ? apiId : undefined,
      start: start ? parseInt(start) : undefined,
      end: end ? parseInt(end) : undefined,
    });
    console.log(verificationsFromAllKeys);
    const verifications: {
      [time: number]: {
        time: number;
        success: number;
        rateLimited: number;
        usageExceeded: number;
      };
    } = {};
    for (const dataPoint of verificationsFromAllKeys.data) {
      if (!verifications[dataPoint.time]) {
        verifications[dataPoint.time] = {
          time: dataPoint.time,
          success: 0,
          rateLimited: 0,
          usageExceeded: 0,
        };
      }

      verifications[dataPoint.time].success += dataPoint.success;
      verifications[dataPoint.time].rateLimited += dataPoint.rateLimited;
      verifications[dataPoint.time].usageExceeded += dataPoint.usageExceeded;
    }

    const apis: {
      [apiId: string]: {
        apiName: string;
        keys: string[];
      };
    } = {};
    authorizedKeys.forEach(({ key, api }) => {
      if (!apis[api.id]) {
        apis[api.id] = {
          apiName: api.name,
          keys: [],
        };
      }
      apis[api.id].keys.push(key.id);
    });

    return c.json({
      ownerId,
      apis: Object.entries(apis).map(([apiId, { apiName, keys }]) => ({
        apiId,
        apiName,
        keys,
      })),
      verificationsByDate: Object.entries(verifications).map(
        ([time, { success, rateLimited, usageExceeded }]) => ({
          time: new Date(parseInt(time)).toISOString().split("T")[0],
          success,
          rateLimited,
          usageExceeded,
        }),
      ),
    });
  });
