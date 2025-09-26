import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { dateTimeToUnix } from "@unkey/clickhouse/src/util";
import { buildUnkeyQuery } from "@unkey/rbac";

const validation = {
  groupBy: z.enum(["key", "identity", "tags", "tag", "month", "day", "hour"]),
};

const route = createRoute({
  deprecated: true,
  tags: ["analytics"],
  operationId: "getVerifications",
  summary: "Get verification analytics",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/analytics.getVerifications",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      apiId: z.string().openapi({
        description: "Select the API. Only keys belonging to this API will be included.",
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
      tag: z
        .string()
        .or(z.array(z.string()))
        .optional()
        .openapi({
          description: `Only include data for a specific tag or tags.

When you are providing zero or more than one tag, all usage counts are aggregated and summed up. Send multiple requests with one tag each if you need counts per tag.`,
          example: ["key_1234"],
        }),
      start: z.coerce
        .number()
        .int()
        .openapi({
          description: `The start of the period to fetch usage for as unix milliseconds timestamp.
To understand how the start filter works, let's look at an example:

You specify a timestamp of 5 minutes past 9 am.
Your timestamp gets truncated to the start of the hour and then applied as filter.
We will include data \`where time >= 9 am\``,
          example: 1620000000000,
        }),
      end: z.coerce
        .number()
        .int()
        .optional()
        .default(() => Date.now())
        .openapi({
          description: `The end of the period to fetch usage for as unix milliseconds timestamp.
To understand how the end filter works, let's look at an example:

You specify a timestamp of 5 minutes past 9 am.
Your timestamp gets truncated to the start of the hour and then applied as filter.
We will include data \`where time <= 10 am\``,
          example: 1620000000000,
        }),
      groupBy: validation.groupBy
        .or(z.array(validation.groupBy))
        .optional()
        .openapi({
          description: `By default, datapoints are not aggregated, however you probably want to get a breakdown per time, key or identity.

Grouping by tags and by tag is mutually exclusive.`,
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
      orderBy: z
        .enum([
          "time",
          "valid",
          "notFound",
          "forbidden",
          "usageExceeded",
          "rateLimited",
          "unauthorized",
          "disabled",
          "insufficientPermissions",
          "expired",
          "total",
        ])
        .optional()
        .openapi({
          description:
            "Sort the output by a specific value. You can use this in combination with the `order` param.",
        }),
      order: z.enum(["asc", "desc"]).optional().default("asc").openapi({
        description: "Define the order of sorting. Use this in combination with `orderBy`",
      }),
    }),
  },
  responses: {
    200: {
      description:
        "Retrieve all required data to build end-user facing dashboards and drive your usage-based billing.",
      content: {
        "application/json": {
          schema: z
            .array(
              z.object({
                time: z.number().int().optional().openapi({
                  description:
                    "Unix timestamp in milliseconds of the start of the current time slice.",
                }),

                valid: z.number().int().optional(),
                notFound: z.number().int().optional(),
                forbidden: z.number().int().optional(),
                usageExceeded: z.number().int().optional(),
                rateLimited: z.number().int().optional(),
                unauthorized: z.number().int().optional(),
                disabled: z.number().int().optional(),
                insufficientPermissions: z.number().int().optional(),
                expired: z.number().int().optional(),
                total: z.number().int().openapi({
                  description:
                    "Total number of verifications in the current time slice, regardless of outcome.",
                }),

                tag: z.string().optional().openapi({
                  description: "Only available when grouping by tag.",
                }),
                tags: z.array(z.string()).optional().openapi({
                  description: "Filter by one or multiple tags. If multiple tags are provided",
                }),
                keyId: z
                  .string()
                  .optional()
                  .openapi({
                    description: `Only available when specifying groupBy=key in the query.
In this case there would be one datapoint per time and groupBy target.`,
                  }),

                identity: z
                  .object({
                    id: z.string(),
                    externalId: z.string(),
                  })
                  .optional()
                  .openapi({
                    description: `Only available when specifying groupBy=identity in the query.
In this case there would be one datapoint per time and groupBy target.`,
                  }),
              }),
            )
            .openapi({
              description:
                "Successful responses will always return an array of datapoints. One datapoint per granular slice, ie: hourly granularity means you receive one element per hour within the queried interval.",
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

    if (filters.start >= filters.end) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "start cannot be equal or greater than end",
      });
    }

    if (
      filters.groupBy &&
      Array.isArray(filters.groupBy) &&
      filters.groupBy.includes("tag") &&
      filters.groupBy.includes("tags")
    ) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "You can not group by tag and tags at the same time",
      });
    }

    const { cache, db, logger, analytics } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("api.*.read_api", `api.${filters.apiId}.read_api`)),
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

    const select = [
      "sumIf(count, outcome == 'VALID') AS valid",
      "sumIf(count, outcome == 'NOT_FOUND') AS notFound",
      "sumIf(count, outcome == 'FORBIDDEN') AS forbidden",
      "sumIf(count, outcome == 'USAGE_EXCEEDED') AS usageExceeded",
      "sumIf(count, outcome == 'RATE_LIMITED') AS rateLimited",
      "sumIf(count, outcome == 'UNAUTHORIZED') AS unauthorized",
      "sumIf(count, outcome == 'DISABLED') AS disabled",
      "sumIf(count, outcome == 'INSUFFICIENT_PERMISSIONS') AS insufficientPermissions",
      "sumIf(count, outcome == 'EXPIRED') AS expired",
      "SUM(count) AS total",
    ];
    const where: string[] = [`workspace_id = '${auth.authorizedWorkspaceId}'`];
    const groupBy: string[] = [];

    type ValueOf<T> = T[keyof T];

    /**
     * By default we use the hourly table, as it is the most accurate.
     * A future optimisation would be to choose a coarser granularity when the
     * requested timeframe is much larger.
     *
     * A user may override this by specifying a groupBy filter
     */
    let table: ValueOf<typeof tables> = tables.hour;
    /**
     * for each groupBy value we add the value manually to prevent SQL injection.
     */

    const selectedGroupBy = (
      Array.isArray(filters.groupBy) ? filters.groupBy : [filters.groupBy]
    ).filter(Boolean);
    if (selectedGroupBy.includes("month")) {
      select.push("time");
      groupBy.push("time");
      table = tables.month;
    } else if (selectedGroupBy.includes("day")) {
      select.push("time");
      groupBy.push("time");
      table = tables.day;
    } else if (selectedGroupBy.includes("hour")) {
      select.push("time");
      groupBy.push("time");
      table = tables.hour;
    }

    const filteredTags = filters.tag
      ? Array.isArray(filters.tag)
        ? filters.tag
        : [filters.tag]
      : undefined;
    if (filteredTags) {
      where.push("AND hasAll(tags, {tags:Array(String)})");
    }

    if (selectedGroupBy.includes("key")) {
      select.push("key_id AS keyId");
      groupBy.push("key_id");
    }
    if (selectedGroupBy.includes("identity")) {
      select.push("identity_id as identityId");
      where.push("AND isNotNull(identity_id)");
      groupBy.push("identity_id");
    }
    if (selectedGroupBy.includes("tags")) {
      select.push("tags");
      groupBy.push("tags");
    }

    if (selectedGroupBy.includes("tag")) {
      select.push("arrayJoin(tags) AS tag");
      groupBy.push("tag");
    }

    const { val: api, err: getApiError } = await cache.apiById.swr(
      filters.apiId,
      async (apiId: string) => {
        return (
          (await db.readonly.query.apis.findFirst({
            where: (table, { eq, and, isNull }) =>
              and(eq(table.id, apiId), isNull(table.deletedAtM)),
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
    where.push(`AND key_space_id = '${api.keyAuthId}'`);

    if (filters.externalId) {
      const { val: identity, err: getIdentityError } = await cache.identityByExternalId.swr(
        `${auth.authorizedWorkspaceId}:${filters.externalId}`,
        async () => {
          return (
            (await db.readonly.query.identities.findFirst({
              where: (table, { and, eq }) =>
                and(
                  eq(table.workspaceId, auth.authorizedWorkspaceId),
                  eq(table.externalId, filters.externalId!),
                ),
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
      where.push(`AND identity_id = '${identity.id}'`);
    }
    const filteredKeyIds = filters.keyId
      ? Array.isArray(filters.keyId)
        ? filters.keyId
        : [filters.keyId]
      : undefined;
    if (filteredKeyIds && filteredKeyIds.length > 0) {
      where.push("AND key_id IN {keyIds:Array(String)}");
    }
    where.push("AND time >= fromUnixTimestamp64Milli({start:Int64})");
    where.push("AND time <= fromUnixTimestamp64Milli({end:Int64})");

    const query: string[] = [];
    query.push(`SELECT ${[...new Set(select)].join(", ")}`);
    query.push(`FROM ${table.name} `);
    query.push(`WHERE ${[...new Set(where)].join("\n")}`);

    if (groupBy.length > 0) {
      query.push(`GROUP BY ${groupBy.join(", ")}`);
    }
    if (filters.orderBy) {
      query.push(`ORDER BY { orderBy: Identifier } ${filters.order === "asc" ? "ASC" : "DESC"} `);
    } else if (groupBy.includes("time")) {
      query.push("ORDER BY time ASC");
    }
    if (filters.limit) {
      query.push("LIMIT {limit: Int64}");
    }

    if (groupBy.includes("time")) {
      query.push(table.fill);
    }

    query.push(";");

    const data = await analytics.internalQuerier.query({
      query: query.map((l) => l.trim()).join("\n"),
      params: z.object({
        start: z.number().int(),
        end: z.number().int(),
        orderBy: z.string().optional(),
        limit: z.number().int().optional(),
        tags: z.array(z.string()).optional(),
        keyIds: z.array(z.string()).optional(),
      }),
      schema: z.object({
        time: dateTimeToUnix.optional(),
        valid: z.number().int().optional(),
        notFound: z.number().int().optional(),
        forbidden: z.number().int().optional(),
        usageExceeded: z.number().int().optional(),
        rateLimited: z.number().int().optional(),
        unauthorized: z.number().int().optional(),
        disabled: z.number().int().optional(),
        insufficientPermissions: z.number().int().optional(),
        expired: z.number().int().optional(),
        total: z.number().int().default(0),
        keyId: z.string().optional(),
        tag: z.string().optional(),
        tags: z.array(z.string()).optional(),
        identityId: z.string().optional(),
      }),
    })({
      start: filters.start,
      end: filters.end,
      orderBy: filters.orderBy,
      limit: filters.limit,
      tags: filteredTags,
      keyIds: filteredKeyIds,
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

    const identityIds: Record<string, boolean> = {};
    for (const row of data.val) {
      if (row.identityId) {
        identityIds[row.identityId] = true;
      }
    }

    const identities = await Promise.all(
      Object.keys(identityIds).map(async (id) => {
        const { val, err } = await cache.identityById.swr(id, () =>
          db.readonly.query.identities.findFirst({
            where: (table, { and, eq }) =>
              and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, id)),
          }),
        );
        if (err) {
          throw new UnkeyApiError({
            code: "INTERNAL_SERVER_ERROR",
            message: "unable to load identity from database",
          });
        }
        return val;
      }),
    );

    const identitiesById = identities.reduce(
      (acc, i) => {
        if (i) {
          acc[i.id] = i;
        }
        return acc;
      },
      {} as Record<string, { externalId: string }>,
    );

    return c.json(
      data.val.map((row) => ({
        time: row.time,
        valid: row.valid,
        notFound: row.notFound,
        forbidden: row.forbidden,
        usageExceeded: row.usageExceeded,
        rateLimited: row.rateLimited,
        unauthorized: row.unauthorized,
        disabled: row.disabled,
        insufficientPermissions: row.insufficientPermissions,
        expired: row.expired,
        total: row.total,
        keyId: row.keyId,
        tag: row.tag,
        tags: row.tags,
        identity: row.identityId
          ? {
              id: row.identityId,
              externalId: identitiesById[row.identityId]?.externalId,
            }
          : undefined,
      })),
    );
  });
