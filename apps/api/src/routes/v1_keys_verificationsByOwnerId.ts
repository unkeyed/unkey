import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import type { Analytics } from "@/pkg/analytics";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { Api } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  method: "get",
  path: "/v1/keys.verificationsByOwnerId",
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
            verifications: z.array(
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
export type V1AnalyticsGetVerificationsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1keysVerificationsByOwnerId = (app: App) =>
  app.openapi(route, async (c) => {
    const { ownerId, apiId, start, end, granularity } = c.req.query();

    if (ownerId === undefined || ownerId === null || ownerId === "") {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "OwnerId must be provided",
      });
    }

    const { analytics, db } = c.get("services");
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
      key: {
        ownerId: string | null;
        name: string | null;
        workspaceId: string;
        enabled: boolean;
        id: string;
        createdAt: Date;
        deletedAt: Date | null;
        keyAuthId: string;
        expires: Date | null;
      };
      api: Api;
    }[] = [];

    //Get all keys by ownerId and optionally apiId

    const keys = async () => {
      const dbRes = await db.query.keys
        .findMany({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.ownerId, ownerId), isNull(table.deletedAt)),
          with: {
            keyAuth: {
              with: {
                api: true,
              },
            },
          },
        })
        .catch((err) => {
          throw new UnkeyApiError({
            code: "INTERNAL_SERVER_ERROR",
            message: `Unbable to load keys by ownerId: ${err.message}`,
          });
        });
      if (!dbRes) {
        return [];
      }

      return dbRes.map((key) => {
        return {
          key: {
            ownerId: key.ownerId,
            name: key.name,
            workspaceId: key.workspaceId,
            enabled: key.enabled,
            id: key.id,
            createdAt: key.createdAt,
            deletedAt: key.deletedAt,
            keyAuthId: key.keyAuthId,
            expires: key.expires,
          },
          api: key.keyAuth.api,
        };
      });
    };

    if (keys.length === 0) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `ownerId ${ownerId} not found`,
      });
    }

    const apiIds = Array.from(new Set(keysList.map(({ api }) => api.id)));

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or, and }) =>
        or(
          "*",
          "api.*.read_key",
          and(...apiIds.map((apiId) => or(`api.${apiId}.read_key`, `api.${apiId}.read_api`))),
        ),
      ),
    );

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const authorizedKeys = keysList.filter(({ api }) => api.workspaceId === authorizedWorkspaceId);

    if (!authorizedKeys) {
      throw new UnkeyApiError({
        code: "FORBIDDEN",
        message: "cannot read keys from a different workspace",
      });
    }

    const verificationsFromAllKeys = await getVerifications({
      workspaceId: authorizedWorkspaceId,
      ownerId: ownerId,
      apiId: apiId ? apiId : undefined,
      start: start ? Number.parseInt(start) : undefined,
      end: end ? Number.parseInt(end) : undefined,
    });
    const verifications: {
      [time: number]: {
        time: number;
        success: number;
        rateLimited: number;
        usageExceeded: number;
      };
    } = {};
    for (const dataPoint of verificationsFromAllKeys.data.sort((a, b) => a.time - b.time)) {
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
      verifications: Object.entries(verifications).map(
        ([time, { success, rateLimited, usageExceeded }]) => ({
          time: new Date(Number.parseInt(time)).toISOString(),
          success,
          rateLimited,
          usageExceeded,
        }),
      ),
    });
  });
