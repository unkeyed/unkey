import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, inArray, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "removePermissions",
  method: "post",
  path: "/v1/keys.removePermissions",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().min(1).openapi({
              description: "The id of the key.",
            }),
            permissions: z
              .array(
                z.object({
                  id: z.string().min(3).optional().openapi({
                    description:
                      "The id of the permission. Provide either `id` or `name`. If both are provided `id` is used.",
                  }),
                  name: z.string().min(1).optional().openapi({
                    description:
                      "Identify the permission via its name. Provide either `id` or `name`. If both are provided `id` is used.",
                  }),
                }),
              )
              .min(1)
              .openapi({
                description: "The permissions you want to remove from this key.",
                example: [
                  {
                    id: "perm_123",
                  },
                  {
                    name: "dns.record.create",
                  },
                ],
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Success",
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
export type V1KeysRemovePermissionsRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysRemovePermissionsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysRemovePermissions = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.remove_permission_from_key")),
    );

    const { db, analytics } = c.get("services");

    const [key, connectedPermissions] = await Promise.all([
      db.primary.query.keys.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, req.keyId)),
      }),

      await db.primary.query.keysPermissions.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, req.keyId)),
        with: {
          permission: true,
        },
      }),
    ]);
    if (!key) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${req.keyId} not found`,
      });
    }

    const deletePermissions = connectedPermissions.filter((cr) => {
      for (const deleteRequest of req.permissions) {
        if ("id" in deleteRequest) {
          return cr.permissionId === deleteRequest.id;
        }
        if ("name" in deleteRequest) {
          return cr.permission.name === deleteRequest.name;
        }
      }
    });

    if (deletePermissions.length === 0) {
      // We have nothing to do
      return c.json({});
    }
    await db.primary.delete(schema.keysPermissions).where(
      and(
        eq(schema.keysPermissions.workspaceId, auth.authorizedWorkspaceId),
        eq(schema.keysPermissions.keyId, key.id),
        inArray(
          schema.keysPermissions.permissionId,
          deletePermissions.map((r) => r.permissionId),
        ),
      ),
    );

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs(
        deletePermissions.map((r) => ({
          workspaceId: auth.authorizedWorkspaceId,
          event: "authorization.disconnect_permission_and_key" as const,
          actor: {
            type: "key" as const,
            id: auth.key.id,
          },
          description: `Disonnected ${r.permissionId} and ${req.keyId}`,
          resources: [
            {
              type: "permission" as const,
              id: r.permissionId,
            },
            {
              type: "key" as const,
              id: req.keyId,
            },
          ],

          context: {
            location: c.get("location"),
            userAgent: c.get("userAgent"),
          },
        })),
      ),
    );

    return c.json({});
  });
