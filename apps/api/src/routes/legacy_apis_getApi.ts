import { cache, db } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildQuery } from "@unkey/rbac";

const route = createRoute({
  method: "get",
  path: "/v1/apis/{apiId}",
  request: {
    header: z.object({
      authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
        description: "A root key to authorize the request formatted as bearer token",
        example: "Bearer unkey_1234",
      }),
    }),
    params: z.object({
      apiId: z.string().min(1).openapi({
        description: "The id of the api to fetch",
        example: "api_1234",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of the key",
              example: "key_1234",
            }),
            workspaceId: z.string().openapi({
              description: "The id of the workspace that owns the api",
              example: "ws_1234",
            }),

            name: z.string().optional().openapi({
              description:
                "The name of the api. This is internal and your users will not see this.",
              example: "Unkey - Production",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type LegacyApisGetApiResponse = z.infer<
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;
export const registerLegacyApisGetApi = (app: App) =>
  app.openapi(route, async (c) => {
    const apiId = c.req.param("apiId");
    const auth = await rootKeyAuth(
      c,
      buildQuery(({ or }) => or("*", `api.${apiId}.read_api`)),
    );

    const api = await cache.withCache(c, "apiById", apiId, async () => {
      return (
        (await db.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
        })) ?? null
      );
    });

    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }

    return c.json({
      id: api.id,
      workspaceId: api.workspaceId,
      name: api.name,
    });
  });
