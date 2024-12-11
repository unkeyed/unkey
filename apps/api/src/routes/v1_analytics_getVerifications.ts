import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["analytics"],
  operationId: "getVerifications",
  method: "get",
  path: "/v1/analytics.getVerifications",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      apiId: z
        .array(z.string())
        .optional()
        .openapi({
          description: `Select the API for which to return data.

        When you are providing zero or more than one API id, all usage counts are aggregated and summed up. Send multiple requests with one apiId each if you need counts per API.
`,
        }),
      externalId: z
        .array(z.string())
        .optional()
        .openapi({
          description: `Filtering by externalId allows you to narrow down the search to a specific user or organisation.

        When you are providing zero or more than one external ids, all usage counts are aggregated and summed up. Send multiple requests with one externalId each if you need counts per identity.

`,
        }),
      keyId: z
        .array(z.string())
        .optional()
        .openapi({
          description: `Only include data for a speciifc key or keys.

        When you are providing zero or more than one key ids, all usage counts are aggregated and summed up. Send multiple requests with one keyId each if you need counts per key.

`,
          example: ["key_1234"],
        }),
      start: z.coerce.number().int().openapi({
        description: "The start of the period to fetch usage for as unix milliseconds timestamp.",
        example: 1620000000000,
      }),
      end: z.coerce.number().int().optional().openapi({
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
                    valid: z.number().int(),
                    rateLimited: z.number().int(),
                    usageExceeded: z.number().int(),
                    total: z.number().int().openapi({
                      description:
                        "Total number of verifications in the current time slice, regardless of outcome.",
                    }),
                    TODO: z.number().int(),
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
export type V1AnalaticsGetVerificationsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1AnalyticsGetVerifications = (app: App) =>
  app.openapi(route, async (c) => {
    const filters = c.req.valid("query");
    console.info("fitlers", filters);

    //    const { analytics, cache, db, logger } = c.get("services");

    // TODO: check permissions
    // const auth = await rootKeyAuth(c, buildUnkeyQuery(({ or }) => or("*")))
    // console.info(auth)

    const query: string[] = [];
    query.push("SELECT *");
    query.push("FROM {table:Identifier}");
    query.push("WHERE workspace_id = {workspaceId: String}");
    if (filters.apiId) {
      // TODO: look up keySpaceId
      // query.push("AND key_space_id = {keySpaceId: String}")
    }
    if (filters.externalId) {
      // TODO: look up identity
      // query.push("AND identity_id = {identityId: String}")
    }
    if (filters.keyId) {
      query.push("AND key_id = {keyId: String}");
    }
    query.push("AND time >= fromUnixTimestamp64Milli({start:Int64})");
    query.push("AND time <= fromUnixTimestamp64Milli({end:Int64})");

    query.push("GROUP BY time, outcome");

    /**
     * for each groupBy value we add the value manually to prevent SQL injection.
     *
     * I think validating this with zod should be enough, but let's use the proper tools to protect
     * ourselves.
     */
    const groupBy = (Array.isArray(filters.groupBy) ? filters.groupBy : [filters.groupBy]).filter(
      Boolean,
    );
    if (groupBy.includes("key")) {
      query.push(", key");
    }
    if (groupBy.includes("identity")) {
      query.push(", identity");
    }
    if (groupBy.includes("tags")) {
      query.push(", tags");
    }

    query.push("ORDER BY {orderBy:Identifier} {order:Identifier}");

    query.push("WITH FILL");
    switch (filters.granularity) {
      case "hour": {
        query.push(
          "FROM toStartOfHour(fromUnixTimestamp64Milli({start: Int64}))",
          "TO toStartOfHour(fromUnixTimestamp64Milli({end: Int64}))",
          "STEP INTERVAL 1 HOUR",
        );
        break;
      }
      case "day": {
        query.push(
          "FROM toStartOfDay(fromUnixTimestamp64Milli({start: Int64}))",
          "TO toStartOfDay(fromUnixTimestamp64Milli({end: Int64}))",
          "STEP INTERVAL 1 DAY",
        );
        break;
      }
      case "month": {
        query.push(
          "FROM toStartOfMonth(fromUnixTimestamp64Milli({start: Int64}))",
          "TO toStartOfMonth(fromUnixTimestamp64Milli({end: Int64}))",
          "STEP INTERVAL 1 Month",
        );
        break;
      }
    }

    console.info("query", query.join("\n"));

    return c.json({ data: [] });
  });
