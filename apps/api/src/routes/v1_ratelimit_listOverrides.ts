import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "listOverrides",
  method: "get",
  path: "/v1/ratelimit.listOverrides",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      // Todo: Refine the descriptions and examples once working
        namespaceId: z.string().optional().openapi({
          description:
            "The id of the namespace. Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
          example: "ns_123",
        }),
        namespaceName: z.string().optional().openapi({
          description:
            "The name of the namespace. Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
          example: "email.outbound",
        }),

      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of keys to return",
        example: 100,
      }),
      cursor: z.string().optional().openapi({
        description:
          "Use this to fetch the next page of results. A new cursor will be returned in the response if there are more results.",
      }),
      limit: z.coerce.number().int().min(1).max(100).optional().default(100).openapi({
        description: "The maximum number of keys to return",
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
      description: "List of overrides for the namespace and optional identifier",
      content: {
        "application/json": {
          schema: z.object({
            overrides: z.array(
              z.object({
                id: z.string(),
                identifier: z.string(),
                limit: z.number().int(),
                duration: z.number().int(),
                async: z.boolean().nullable().optional(),
              }),
            ),
            cursor: z.string().optional().openapi({
              description:
                "The cursor to use for the next page of results, if no cursor is returned, there are no more results",
              example: "eyJrZXkiOiJrZXlfMTIzNCJ9",
            }),
            total: z.number().int().openapi({
              description: "The total number of overrides for the namespace",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1RatelimitListOverridesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export type V1RatelimitListOverridesRequest = z.infer<
  (typeof route.request)["query"]
>;
export const registerV1RatelimitListOverrides = (app: App) =>
  app.openapi(route, async (c) => {

    const { namespaceId, namespaceName, limit, cursor } = c.req.valid("query");
    const { db } = c.get("services");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.read_override")),
    );
    if(!auth.authorizedWorkspaceId) {
      throw new Error("No authorized workspace found");
    }
    

    if(!namespaceId && !namespaceName) {
      throw new Error("Either id or name must be provided");
    }
    // Do we want to add a cache for this like on keys?
    const foundNSpace = await db.primary.query.ratelimitNamespaces.findFirst({
      where: (table, { eq, and }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          namespaceId ? eq(table.id, namespaceId) : eq(table.name, namespaceName!),
        ),
       
    });
    if(!foundNSpace) {
      return c.json({});
    }
    // Change this from Identity to RatelimitOverride
    const [overrides, total] = await Promise.all([
      db.readonly.query.ratelimitOverrides.findMany({
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


//Old Please change
    // const overrides = await db.primary.query.ratelimitOverrides.findMany({
    //   where: (table, { eq, and }) =>
    //     and(
    //       eq(table.workspaceId, auth.authorizedWorkspaceId),
    //       eq(table.namespaceId, namespaceId),
    //       eq(table.identifier, identifier),
    //     ),
    //   take: limit,
    //   cursor: cursor ? { id: cursor } : undefined,
    //   orderBy: { id: "asc" },
    // });

    return c.json({
      overrides: overrides.map((r) => ({
        id: r.id,
        identifier: r.identifier,
        limit: r.limit,
        duration: r.duration,
        async: r.async ?? undefined,
      })),
      total: overrides.length,
      cursor: overrides.at(-1)?.id ?? undefined,
    });
  });
