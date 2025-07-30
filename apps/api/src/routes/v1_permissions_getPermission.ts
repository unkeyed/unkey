import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "getPermission",
  summary: "Get permission",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
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
      description: "The Role",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of the permission",
              example: "perm_123",
            }),

            name: z.string().openapi({
              description: "The name of the permission.",
              example: "domain.record.manager",
            }),
            slug: z.string().openapi({
              description: "The slug of the permission",
              example: "domain-record-manager",
            }),
            description: z.string().optional().openapi({
              description:
                "The description of what this permission does. This is just for your team, your users will not see this.",
              example: "Can manage dns records",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1PermissionsGetPermissionResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1PermissionsGetPermission = (app: App) =>
  app.openapi(route, async (c) => {
    const { permissionId } = c.req.valid("query");
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.read_permission")),
    );

    const permission = await db.readonly.query.permissions.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, permissionId)),
    });
    if (!permission) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `permission ${permissionId} not found`,
      });
    }

    return c.json({
      id: permission.id,
      name: permission.name,
      slug: permission.slug,
      description: permission.description ?? undefined,
    });
  });
