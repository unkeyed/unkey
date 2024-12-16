import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
// import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { dateTimeToUnix } from "@unkey/clickhouse/src/util";
import { buildUnkeyQuery } from "@unkey/rbac";
// import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["analytics"],
  operationId: "getVerifications",
  method: "get",
  path: "/v1/analytics.getVerifications",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      apiId: z
        .string()
        .optional()
        .openapi({
          description: `Select the API for which to return data.

        When you are providing zero or more than one API id, all usage counts are aggregated and summed up. Send multiple requests with one apiId each if you need counts per API.
`,
        }),
      externalId: z.string().optional().openapi({
        description:
          "Filtering by externalId allows you to narrow down the search to a specific user or organisation.",
      }),
      keyId: z
        .string()
        .or(z.array(z.string()))
        .optional()
        .openapi({
          description: `Only include data for a specific key or keys.

        When you are providing zero or more than one key ids, all usage counts are aggregated and summed up. Send multiple requests with one keyId each if you need counts per key.

`,
          example: ["key_1234"],
        }),
      start: z.coerce.number().int().openapi({
        description: "The start of the period to fetch usage for as unix milliseconds timestamp.",
        example: 1620000000000,
      }),
      end: z.coerce
        .number()
        .int()
        .optional()
        .default(() => Date.now())
        .openapi({
          description:
            "The end of the period to fetch usage for as unix milliseconds timestamp, defaults to now.",
          example: 1620000000000,
        }),
      granularity: z.enum(["hour", "day", "month"]).openapi({
        description:
          "Selects the granularity of data. For example selecting hour will return one datapoint per hour.",
        example: "day",
      }),
      groupBy: z
        .enum(["key", "identity", "tags"])
        .or(z.array(z.enum(["key", "identity", "tags"])))
        .optional()
        .openapi({
          description: `By default, all datapoints are aggregated by time alone, summing up all verifications across identities and keys. However in certain scenarios you want to get a breakdown per key or identity. For example finding out the usage spread across all keys for a specific user.


`,
        }),
      limit: z.coerce
        .number()
        .int()
        .min(1)
        .optional()
        .openapi({
          description: `Limit the number of returned datapoints.
        This may become useful for querying the top 10 identities based on usage.`,
        }),
      orderBy: z.enum(["total", "valid", "TODO"]).optional().openapi({
        description: "TODO",
      }),
      order: z.enum(["asc", "desc"]).optional().default("asc").openapi({
        description: "TODO",
      }),
    }),
  },
  responses: {
    200: {
      description:
        "Retrieve all required data to build end-user facing dashboards and drive your usage-based billing.",
      content: {
        "application/json": {
          schema: z.object({
            data: z
              .array(
                z.object({
                  time: z.number().int().openapi({
                    description:
                      "Unix timestamp in milliseconds of the start of the current time slice.",
                  }),

                  outcomes: z.object({
                    valid: z.number().int().optional(),
                    notFound: z.number().int().optional(),
                    forbidden: z.number().int().optional(),
                    usageExceeded: z.number().int().optional(),
                    rateLimited: z.number().int().optional(),
                    unauthorited: z.number().int().optional(),
                    disabled: z.number().int().optional(),
                    insufficientPermissions: z.number().int().optional(),
                    expired: z.number().int().optional(),
                    total: z.number().int().openapi({
                      description:
                        "Total number of verifications in the current time slice, regardless of outcome.",
                    }),
                  }),

                  keyId: z
                    .string()
                    .optional()
                    .openapi({
                      description: `
                Only available when specifying groupBy=key in the query.
                In this case there would be one datapoint per time and groupBy target.`,
                    }),
                  apiId: z
                    .string()
                    .optional()
                    .openapi({
                      description: `
                Only available when specifying groupBy=api in the query.
                In this case there would be one datapoint per time and groupBy target.`,
                    }),
                  identity: z
                    .object({
                      id: z.string(),
                      externalId: z.string(),
                    })
                    .optional()
                    .openapi({
                      description: `
                Only available when specifying groupBy=identity in the query.
                In this case there would be one datapoint per time and groupBy target.`,
                    }),
                }),
              )
              .openapi({
                description:
                  "Successful responses will always return an array of datapoints. One datapoint per granular slice, ie: hourly granularity means you receive one element per hour within the queried interval.",
              }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1AnalyticsGetVerificationsRequest = z.infer<typeof route.request.query>;

export type V1AnalyticsGetVerificationsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1AnalyticsGetVerifications = (app: App) =>
  app.openapi(route, async (c) => {
    const filters = c.req.valid("query");
    console.info("fitlers", filters);

    const { cache, db, logger, analytics } = c.get("services");

    // TODO: check permissions
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("api.*.read_api")),
    );

    const tables = {
      hour: {
        name: "verifications.key_verifications_per_hour_v3",
        fill: `WITH FILL
          FROM toStartOfHour(fromUnixTimestamp64Milli({ start: Int64 }))
          TO toStartOfHour(fromUnixTimestamp64Milli({ end: Int64 }))
          STEP INTERVAL 1 HOUR`,
      },
      day: {
        name: "verifications.key_verifications_per_day_v3",
        fill: `WITH FILL
          FROM toStartOfDay(fromUnixTimestamp64Milli({ start: Int64 }))
          TO toStartOfDay(fromUnixTimestamp64Milli({ end: Int64 }))
          STEP INTERVAL 1 DAY`,
      },
      month: {
        name: "verifications.key_verifications_per_month_v3",
        fill: `WITH FILL
          FROM toDateTime(toStartOfMonth(fromUnixTimestamp64Milli({ start: Int64 })))
          TO toDateTime(toStartOfMonth(fromUnixTimestamp64Milli({ end: Int64 })))
          STEP INTERVAL 1 MONTH`,
      },
    } as const;

    const table = tables[filters.granularity];

    const select = [
      "time",
      `map(
        'valid', sumIf(count, outcome=='VALID'),
        'notFound', sumIf(count, outcome=='NOT_FOUND'),
        'forbidden', sumIf(count, outcome=='FORBIDDEN'),
        'usageExceeded', sumIf(count, outcome=='USAGE_EXCEEDED'),
        'rateLimited', sumIf(count, outcome=='RATE_LIMITED'),
        'unauthorited', sumIf(count, outcome=='UNAUTHORIZED'),
        'disabled', sumIf(count, outcome=='DISABLED'),
        'insufficientPermissions', sumIf(count, outcome=='INSUFFICIENT_PERMISSIONS'),
        'expired', sumIf(count, outcome=='EXPIRED'),
        'total', SUM(count)

      ) AS outcomes
      `,
    ];
    const groupBy = ["time"];

    /**
     * for each groupBy value we add the value manually to prevent SQL injection.
     */
    const selectedGroupBy = (
      Array.isArray(filters.groupBy) ? filters.groupBy : [filters.groupBy]
    ).filter(Boolean);
    if (selectedGroupBy.includes("key")) {
      select.push("key_id");
      groupBy.push("key_id");
    }
    if (selectedGroupBy.includes("identity")) {
      select.push("identity_id");
      groupBy.push("identity_id");
    }
    if (selectedGroupBy.includes("tags")) {
      select.push("tags");
      groupBy.push("tags");
    }

    const query: string[] = [];
    query.push(`SELECT ${select.join(", ")}`);
    query.push(`FROM ${table.name}`);
    query.push(`WHERE workspace_id = '${auth.authorizedWorkspaceId}'`);

    if (filters.apiId) {
      const { val: api, err: getApiError } = await cache.apiById.swr(
        filters.apiId,
        async (apiId: string) => {
          return (
            (await db.readonly.query.apis.findFirst({
              where: (table, { eq, and, isNull }) =>
                and(eq(table.id, apiId), isNull(table.deletedAt)),
              with: {
                keyAuth: true,
              },
            })) ?? null
          );
        },
      );
      if (getApiError) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "we're unable to load the API",
        });
      }
      if (!api) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: "we're unable to find the API",
        });
      }
      if (!api.keyAuthId) {
        throw new UnkeyApiError({
          code: "PRECONDITION_FAILED",
          message: "api has no keyspace attached",
        });
      }
      query.push(`AND key_space_id = '${api.keyAuthId}'`);
    }
    if (filters.externalId) {
      const { val: identity, err: getIdentityError } = await cache.identityByExternalId.swr(
        filters.externalId,
        async (externalId: string) => {
          return (
            (await db.readonly.query.identities.findFirst({
              where: (table, { eq }) => eq(table.externalId, externalId),
            })) ?? null
          );
        },
      );
      if (getIdentityError) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "we're unable to load the identity",
        });
      }
      if (!identity) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: "we're unable to find the identity",
        });
      }
      query.push(`AND identity_id = '${identity.id}'`);
    }
    if (filters.keyId) {
      query.push("AND key_id = {keyId: String}");
    }
    query.push("AND time >= fromUnixTimestamp64Milli({start:Int64})");
    query.push("AND time <= fromUnixTimestamp64Milli({end:Int64})");

    query.push(`GROUP BY ${groupBy.join(", ")}`);

    query.push(`ORDER BY { orderBy: Identifier } ${filters.order === "asc" ? "ASC" : "DESC"}`);

    query.push(table.fill);

    query.push(";");

    console.info("query", query.map((l) => l.trim()).join("\n"));

    const data = await analytics.internalQuerier.query({
      query: query.map((l) => l.trim()).join("\n"),
      params: z.object({
        start: z.number().int(),
        end: z.number().int(),
        orderBy: z.string(),
      }),
      schema: z
        .object({
          time: dateTimeToUnix,
          outcomes: z.object({
            valid: z.number().int().optional(),
            notFound: z.number().int().optional(),
            forbidden: z.number().int().optional(),
            usageExceeded: z.number().int().optional(),
            rateLimited: z.number().int().optional(),
            unauthorited: z.number().int().optional(),
            disabled: z.number().int().optional(),
            insufficientPermissions: z.number().int().optional(),
            expired: z.number().int().optional(),
            total: z.number().int().default(0),
          }),
          key_id: z.string().optional(),
        })
        .transform((d) => ({
          ...d,
          time: d.time,
          outcomes: d.outcomes,
          keyId: d.key_id,
        })),
    })({
      start: filters.start,
      end: filters.end,
      orderBy: filters.orderBy ?? "time",
    });

    if (data.err) {
      logger.error("unable to query clickhouse", {
        error: data.err.message,
        query: query,
      });
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "unable to query clickhouse",
      });
    }

    return c.json({ data: data.val });
  });
