import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "getRole",
  summary: "Get role",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/permissions.getRole",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      roleId: z.string().min(1).openapi({
        description: "The id of the role to fetch",
        example: "role_123",
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
              description: "The id of the role",
              example: "role_1234",
            }),

            name: z.string().openapi({
              description: "The name of the role.",
              example: "domain.record.manager",
            }),
            description: z.string().optional().openapi({
              description:
                "The description of what this role does. This is just for your team, your users will not see this.",
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
export type V1PermissionsGetRoleResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1PermissionsGetRole = (app: App) =>
  app.openapi(route, async (c) => {
    const { roleId } = c.req.valid("query");
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.read_role")),
    );

    const role = await db.readonly.query.roles.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, roleId)),
    });
    if (!role) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `role ${roleId} not found`,
      });
    }

    return c.json({
      id: role.id,
      name: role.name,
      description: role.description ?? undefined,
    });
  });
