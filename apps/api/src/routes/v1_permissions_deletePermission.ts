import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["permissions"],
  operationId: "deletePermission",
  method: "post",
  path: "/v1/permissions.deletePermission",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            permissionId: z.string().openapi({
              description: "The id of the permission you want to delete.",
              example: "perm_123",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Sucessfully deleted a permission",
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
export type V1PermissionsDeletePermissionRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1PermissionsDeletePermissionResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1PermissionsDeletePermission = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.delete_permission")),
    );

    const { db, analytics } = c.get("services");

    const permission = await db.primary.query.permissions.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, req.permissionId)),
    });
    if (!permission) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `Permission ${req.permissionId} not found`,
      });
    }

    await db.primary
      .delete(schema.permissions)
      .where(
        and(
          eq(schema.permissions.workspaceId, auth.authorizedWorkspaceId),
          eq(schema.permissions.id, req.permissionId),
        ),
      );

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs({
        workspaceId: auth.authorizedWorkspaceId,
        event: "permission.delete",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Deleted ${permission.id}`,
        resources: [
          {
            type: "permission",
            id: permission.id,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      }),
    );

    return c.json({});
  });
