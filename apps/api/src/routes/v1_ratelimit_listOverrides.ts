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
      namespaceId: z.string().openapi({
        description:
          "Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
        example: "email.outbound",
      }),
      identifier: z.string().optional().openapi({
        description:
          "Identifier of your user, this can be their userId, an email, an ip or anything else. Wildcards ( * ) can be used to match multiple identifiers, More info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
        example: "user_123",
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
      description: "List of overrides for the api",
      content: {
        "application/json": {
          schema: z.object({
            overrides: z.array(
              z.object({
                id: z.string(),
                namespace: z.string(),
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
              description: "The total number of keys for this api",
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
export const registerV1RatelimitListOverrides = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");
    const { namespaceId, identifier } = c.req.valid("query");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.read_override")),
    );

    
    if (!identifier) {
      const overrides = await db.readonly.query.ratelimitOverrides.findMany({
        where: (table, { eq, and }) =>
          and(
            eq(table.workspaceId, auth.authorizedWorkspaceId),
            eq(table.namespaceId, namespaceId),
          ),
        with: {
          namespace: true,
        },
      });

      return c.json({
        overrides: overrides.map((r) => ({
          id: r.id,
          namespace: r.namespace.name,
          identifier: r.identifier,
          limit: r.limit,
          duration: r.duration,
          async: r.async,
        })),
        total: overrides.length,
        cursor: overrides.at(-1)?.id ?? undefined,
      });
    }
    const overrides = await db.readonly.query.ratelimitOverrides.findMany({
      where: (table, { eq, and }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          eq(table.namespaceId, namespaceId),
          eq(table.identifier, identifier),
        ),
      with: {
        namespace: true,
      },
    });

    return c.json({
      overrides: overrides.map((r) => ({
        id: r.id,
        namespace: r.namespace.name,
        identifier: r.identifier,
        limit: r.limit,
        duration: r.duration,
        async: r.async ?? undefined,
      })),
      total: overrides.length,
      cursor: overrides.at(-1)?.id ?? undefined,
    });
  });
