import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
// import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";
// import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["permissions"],
  operationId: "addRolesToKey",
  method: "post",
  path: "/v1/permissions.addRolesToKey",
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
            roles: z
              .array(
                z.union([
                  z.object({
                    id: z.string().min(3).openapi({
                      description: "The id of the role.",
                    }),
                  }),
                  z.object({
                    name: z.string().openapi({
                      description: "The name of the role",
                    }),
                    create: z.boolean().optional().openapi({
                      description:
                        "Set to true to automatically create the role if it does not yet exist.",
                    }),
                  }),
                ]),
              )
              .min(1)
              .openapi({
                description: "The roles you want to add to this key",
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "All currently connected roles",
      content: {
        "application/json": {
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the role. This is used internally",
                example: "role_123",
              }),
              name: z.string().openapi({
                description: "The name of the role",
                example: "dns.record.create",
              }),
            }),
          ),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1PermissionsAddRolesToKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1PermissionsAddRolesToKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1PermissionsAddRolesToKey = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "permission.*.add_role_to_key")),
    );

    const { db, analytics, rbac } = c.get("services");

    const requestedIds = req.roles.filter((r) => "id" in r).map((r) => r.id);
    const requestedNames = req.roles.filter((r) => "name" in r).map((r) => r.name);

    const [existingRoles, connectedRoles] = await Promise.all([
      db.primary.query.roles.findMany({
        where: (table, { eq, or, and, inArray }) =>
          and(
            eq(table.workspaceId, auth.authorizedWorkspaceId),
            or(
              requestedIds.length > 0 ? inArray(table.id, requestedIds) : undefined,
              requestedNames.length > 0 ? inArray(table.name, requestedNames) : undefined,
            ),
          ),
      }),
      await db.primary.query.keysRoles.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, req.keyId)),
      }),
    ]);

    const missingRoleNames: string[] = [];
    for (const role of req.roles) {
      if ("id" in role && !existingRoles.some((ep) => ep.id === role.id)) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `role ${role.id} not found`,
        });
      }
      if ("name" in role && !existingRoles.some((ep) => ep.name === role.name)) {
        if (!role.create) {
          throw new UnkeyApiError({
            code: "NOT_FOUND",
            message: `role ${role.name} not found`,
          });
        }
        missingRoleNames.push(role.name);
      }
    }

    const createRoles = missingRoleNames.map((name) => ({
      id: newId("permission"),
      workspaceId: auth.authorizedWorkspaceId,
      name,
    }));
    if (createRoles.length > 0) {
      const rbacResp = rbac.evaluatePermissions(
        buildUnkeyQuery(({ or }) => or("*", "permission.*.create_permission")),
        auth.permissions,
      );
      if (rbacResp.err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "unable to evaluate permissions",
        });
      }
      if (!rbacResp.val.valid) {
        throw new UnkeyApiError({
          code: "UNAUTHORIZED",
          message: rbacResp.val.message,
        });
      }

      await db.primary.insert(schema.roles).values(createRoles);
    }
    const allRoles = [
      ...existingRoles.map((p) => ({ id: p.id, name: p.name })),
      ...createRoles.map((p) => ({ id: p.id, name: p.name })),
    ];

    const addRoles = allRoles.filter((ar) => !connectedRoles.some((cp) => cp.roleId === ar.id));
    if (addRoles.length > 0) {
      const rbacResp = rbac.evaluatePermissions(
        buildUnkeyQuery(({ or }) => or("*", "permission.*.add_role_to_key")),
        auth.permissions,
      );
      if (rbacResp.err) {
        throw new UnkeyApiError({
          code: "INTERNAL_SERVER_ERROR",
          message: "unable to evaluate permissions",
        });
      }
      if (!rbacResp.val.valid) {
        throw new UnkeyApiError({
          code: "UNAUTHORIZED",
          message: rbacResp.val.message,
        });
      }
      await db.primary.insert(schema.keysRoles).values(
        addRoles.map((r) => ({
          keyId: req.keyId,
          roleId: r.id,
          workspaceId: auth.authorizedWorkspaceId,
        })),
      );
    }

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs([
        ...createRoles.map((r) => ({
          workspaceId: auth.authorizedWorkspaceId,
          event: "role.create" as const,
          actor: {
            type: "key" as const,
            id: auth.key.id,
          },
          description: `Created ${r.id}`,
          resources: [
            {
              type: "role" as const,
              id: r.id,
              meta: {
                name: r.name,
              },
            },
          ],

          context: {
            location: c.get("location"),
            userAgent: c.get("userAgent"),
          },
        })),
        ...addRoles.map((r) => ({
          workspaceId: auth.authorizedWorkspaceId,
          event: "authorization.connect_role_and_key" as const,
          actor: {
            type: "key" as const,
            id: auth.key.id,
          },
          description: `Connected ${r.id} and ${req.keyId}`,
          resources: [
            {
              type: "role" as const,
              id: r.id,
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
      ]),
    );

    return c.json(allRoles.map((p) => ({ id: p.id, name: p.name })));
  });
