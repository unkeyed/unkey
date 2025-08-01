import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, gt, sql } from "@unkey/db";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["identities"],
  operationId: "listIdentities",
  summary: "List identities",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/identities.listIdentities",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      environment: z.string().optional().default("default").openapi({
        description: "This is not yet used but here for future compatibility.",
      }),
      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of identities to return",
        example: 100,
      }),
      cursor: z.string().optional().openapi({
        description:
          "Use this to fetch the next page of results. A new cursor will be returned in the response if there are more results.",
      }),
    }),
  },
  responses: {
    200: {
      description: "A list of identities.",
      content: {
        "application/json": {
          schema: z.object({
            identities: z
              .array(
                z.object({
                  id: z.string().openapi({
                    description: "The id of this identity. Used to interact with unkey's API",
                  }),
                  externalId: z.string().openapi({
                    description: "The id in your system",
                  }),
                  ratelimits: z
                    .array(
                      z.object({
                        name: z.string().openapi({
                          description:
                            "The name of this limit. You will need to use this again when verifying a key.",
                          example: "tokens",
                        }),
                        limit: z.number().int().min(1).openapi({
                          description:
                            "How many requests may pass within a given window before requests are rejected.",
                          example: 10,
                        }),
                        duration: z.number().int().min(1000).openapi({
                          description: "The duration for each ratelimit window in milliseconds.",
                          example: 1000,
                        }),
                      }),
                    )
                    .openapi({
                      description:
                        "When verifying keys, you can specify which limits you want to use and all keys attached to this identity, will share the limits.",
                    }),
                }),
              )
              .openapi({
                description: "A list of identities.",
              }),
            cursor: z.string().optional().openapi({
              description:
                "The cursor to use for the next page of results, if no cursor is returned, there are no more results",
              example: "eyJrZXkiOiJrZXlfMTIzNCJ9",
            }),
            total: z.number().int().openapi({
              description: "The total number of identities for this environment",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
  "x-speakeasy-pagination": {
    type: "cursor",
    inputs: [
      {
        name: "cursor",
        in: "parameters",
        type: "cursor",
      },
    ],
    outputs: {
      nextCursor: "$.cursor",
    },
  },
});

export type Route = typeof route;
export type V1IdentitiesListIdentitiesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1IdentitiesListIdentities = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("query");
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) =>
        or("*", "identity.*.read_identity", `identity.${req.environment}.read_identity`),
      ),
    );

    const [identities, total] = await Promise.all([
      db.readonly.query.identities.findMany({
        where: and(
          ...[
            eq(schema.identities.workspaceId, auth.authorizedWorkspaceId),
            eq(schema.identities.environment, req.environment),
            req.cursor ? gt(schema.keys.id, req.cursor) : undefined,
          ].filter(Boolean),
        ),
        with: {
          ratelimits: true,
        },
        limit: req.limit,
        orderBy: schema.identities.id,
      }),

      db.readonly
        .select({ count: sql<string>`count(*)` })
        .from(schema.identities)
        .where(
          and(
            eq(schema.identities.workspaceId, auth.authorizedWorkspaceId),
            eq(schema.identities.environment, req.environment),
          ),
        )
        .then((rows) => Number(rows.at(0)?.count ?? 0)),
    ]);

    return c.json({
      identities: identities.map((i) => ({
        id: i.id,
        externalId: i.externalId,
        ratelimits: i.ratelimits.map((r) => ({
          name: r.name,
          limit: r.limit,
          duration: r.duration,
        })),
      })),
      total,
      cursor: identities.at(-1)?.id ?? undefined,
    });
  });
