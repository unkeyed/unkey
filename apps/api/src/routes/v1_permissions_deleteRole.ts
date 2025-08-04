import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "deleteRole",
  summary: "Delete role",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/permissions.deleteRole",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            roleId: z.string().openapi({
              description: "The id of the role you want to delete.",
              example: "role_123",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Sucessfully deleted a role",
      content: {
        "application/json": {
          schema: z.object({}),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1PermissionsDeleteRoleRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1PermissionsDeleteRoleResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1PermissionsDeleteRole = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.delete_role")),
    );

    const { db } = c.get("services");

    const role = await db.primary.query.roles.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, req.roleId)),
    });
    if (!role) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `Role ${req.roleId} not found`,
      });
    }

    await db.primary.transaction(async (tx) => {
      await tx
        .delete(schema.roles)
        .where(
          and(
            eq(schema.roles.workspaceId, auth.authorizedWorkspaceId),
            eq(schema.roles.id, req.roleId),
          ),
        );

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: auth.authorizedWorkspaceId,
        event: "role.delete",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Deleted ${role.id}`,
        resources: [
          {
            type: "role",
            id: role.id,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    return c.json({});
  });
