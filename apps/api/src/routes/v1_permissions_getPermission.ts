import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["permissions"],
  operationId: "getPermission",
  method: "get",
  path: "/v1/permissions.getPermission",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      permissionId: z.string().min(1).openapi({
        description: "The id of the permission to fetch",
        example: "perm_123",
      }),
    }),
  },
  responses: {
    200: {
      description: "The permission",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of the key",
              example: "key_1234",
            }),

            name: z.string().openapi({
              description: "The name of the permission.",
              example: "domain.unkey_com.create_record",
            }),
            description: z.string().optional().openapi({
              description:
                "The description of what this permission does. This is just for your team, your users will not see this.",
              example: "domain.unkey_com.create_record",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1ApisGetApiResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1ApisGetApi = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId } = c.req.query();
    const { cache, db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "permission.*.read_permission")),
    );

    const { val: api, err } = await cache.apiById.swr(apiId, async () => {
      return (
        (await db.readonly.query.apis.findFirst({
          where: (table, { eq, and, isNull }) => and(eq(table.id, apiId), isNull(table.deletedAt)),
          with: {
            keyAuth: true,
          },
        })) ?? null
      );
    });
    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to get api: ${err.message}`,
      });
    }
    if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `api ${apiId} not found` });
    }

    return c.json({
      id: api.id,
      workspaceId: api.workspaceId,
      name: api.name,
    });
  });
