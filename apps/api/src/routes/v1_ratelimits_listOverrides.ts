import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, gt, isNull, schema, sql } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["ratelimits"],
  operationId: "listOverrides",
  summary: "List rate limit overrides",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/ratelimits.listOverrides",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      // Todo: Refine the descriptions and examples once working
      namespaceId: z.string().optional().openapi({
        description:
          "The id of the namespace. Either namespaceId or namespaceName must be provided",
        example: "rlns_1234",
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
    }),
  },
  responses: {
    200: {
      description: "List of overrides for the given namespace.",
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
export type V1RatelimitListOverridesRequest = z.infer<(typeof route.request)["query"]>;
export const registerV1RatelimitListOverrides = (app: App) =>
  app.openapi(route, async (c) => {
    const { namespaceId, namespaceName, limit, cursor } = c.req.valid("query");
    const { db } = c.get("services");
    if (!namespaceId && !namespaceName) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "You must provide a namespaceId or a namespaceName",
      });
    }
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.read_override")),
    );
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    if (!authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "Missing required permission: ratelimit.*.read_override",
      });
    }

    const namespace = await db.readonly.query.ratelimitNamespaces.findFirst({
      where: (table, { and, eq }) =>
        and(
          eq(table.workspaceId, authorizedWorkspaceId),
          namespaceId ? eq(table.id, namespaceId) : eq(table.name, namespaceName!),
        ),
    });
    if (!namespace) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `Namespace ${namespaceId ? namespaceId : namespaceName} not found`,
      });
    }

    const [overrides, total] = await Promise.all([
      db.readonly.query.ratelimitOverrides.findMany({
        where: (table, { and, eq }) =>
          and(
            ...[
              isNull(schema.ratelimitOverrides.deletedAtM),
              eq(table.workspaceId, authorizedWorkspaceId),
              eq(table.namespaceId, namespace.id),
              cursor ? gt(schema.ratelimitOverrides.id, cursor) : undefined,
            ].filter(Boolean),
          ),
        limit: limit,
        orderBy: schema.ratelimitOverrides.id,
      }),

      db.readonly
        .select({ count: sql<string>`count(*)` })
        .from(schema.ratelimitOverrides)
        .where(
          and(
            eq(schema.ratelimitOverrides.namespaceId, namespace?.id),
            isNull(schema.ratelimitOverrides.deletedAtM),
          ),
        ),
    ]);
    return c.json({
      overrides:
        overrides.map((k) => ({
          id: k.id,
          identifier: k.identifier,
          limit: k.limit,
          duration: k.duration,
          async: k.async ?? undefined,
        })) ?? [],
      total: Number(total.at(0)?.count ?? 0),
      cursor: overrides.at(-1)?.id ?? undefined,
    });
  });
