import type { App, Context } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, inArray, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "setPermissions",
  method: "post",
  path: "/v1/keys.setPermissions",
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
                z.union([
                  z.object({
                    id: z.string().min(3).openapi({
                      description: "The id of the permission.",
                    }),
                  }),
                  z.object({
                    name: z.string().openapi({
                      description: "The name of the permission",
                    }),
                    create: z.boolean().optional().openapi({
                      description:
                        "Set to true to automatically create the permission if it does not yet exist.",
                    }),
                  }),
                ]),
              )
              .min(1)
              .openapi({
                description: `The permissions you want to set for this key. This overwrites all existing permissions.

You can either define permission ids if you know them, or the permission name. If the name does not exist, an error is returned unless you set \`create: true\`                  `,
                examples: [
                  [
                    {
                      id: "perm_123",
                    },
                    {
                      name: "dns.record.create",
                    },
                  ],
                  [
                    {
                      name: "dns.record.create",
                      create: true,
                    },
                  ],
                ],
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "All currently connected permissions",
      content: {
        "application/json": {
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the permission. This is used internally",
                example: "perm_123",
              }),
              name: z.string().openapi({
                description: "The name of the permission",
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
export type V1KeysSetPermissionsRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysSetPermissionsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysSetPermissions = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(c);

    const allPermissions = await setPermissions(c, auth, req.keyId, {
      ids: req.permissions.filter((r) => "id" in r).map((r) => r.id),
      names: req.permissions.filter((r) => "name" in r),
    });

    return c.json(allPermissions);
  });

export async function setPermissions(
  c: Context,
  auth: {
    authorizedWorkspaceId: string;
    permissions: string[];
    key: { id: string };
  },
  keyId: string,
  requested: {
    ids: Array<string>;
    names: Array<{ name: string; create?: boolean }>;
  },
): Promise<Array<{ id: string; name: string }>> {
  const { db, analytics, cache, rbac } = c.get("services");

  const [key, existingPermissions, connectedPermissions] = await Promise.all([
    db.primary.query.keys.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, keyId)),
    }),
    db.primary.query.permissions.findMany({
      where: (table, { eq, or, and, inArray }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          or(
            requested.ids.length > 0 ? inArray(table.id, requested.ids) : undefined,
            requested.names.length > 0
              ? inArray(
                  table.name,
                  requested.names.map((n) => n.name),
                )
              : undefined,
          ),
        ),
    }),
    await db.primary.query.keysPermissions.findMany({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, keyId)),
      with: {
        permission: {
          columns: {
            name: true,
          },
        },
      },
    }),
  ]);

  if (!key) {
    throw new UnkeyApiError({
      code: "NOT_FOUND",
      message: `key ${keyId} not found`,
    });
  }

  const disconnectPermissions = connectedPermissions.filter((r) => {
    if (requested.ids.includes(r.permissionId)) {
      return false;
    }
    if (requested.names.some(({ name }) => name === r.permission.name)) {
      return false;
    }
    return true;
  });

  if (disconnectPermissions.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.remove_permission_from_key")),
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

    await db.primary.delete(schema.keysPermissions).where(
      and(
        eq(schema.keysPermissions.workspaceId, auth.authorizedWorkspaceId),
        eq(schema.keysPermissions.keyId, key.id),
        inArray(
          schema.keysPermissions.permissionId,
          disconnectPermissions.map((r) => r.permissionId),
        ),
      ),
    );
  }

  const missingPermissionNames: string[] = [];
  for (const id of requested.ids) {
    if (!existingPermissions.some((r) => r.id === id)) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `permission ${id} not found`,
      });
    }
  }
  for (const { create, name } of requested.names) {
    if (!existingPermissions.some((r) => r.name === name)) {
      if (!create) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `permission ${name} not found and not allowed to create`,
        });
      }
      missingPermissionNames.push(name);
    }
  }

  const createPermissions = missingPermissionNames.map((name) => ({
    id: newId("permission"),
    workspaceId: auth.authorizedWorkspaceId,
    name,
  }));
  if (createPermissions.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_permission")),
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
        code: "INSUFFICIENT_PERMISSIONS",
        message: rbacResp.val.message,
      });
    }

    await db.primary.insert(schema.permissions).values(createPermissions);
  }
  const allPermissions = [
    ...existingPermissions.map((p) => ({ id: p.id, name: p.name })),
    ...createPermissions.map((p) => ({ id: p.id, name: p.name })),
  ];

  const addPermissions = allPermissions.filter(
    (ar) => !connectedPermissions.some((cp) => cp.permissionId === ar.id),
  );
  if (addPermissions.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.add_permission_to_key")),
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
        code: "INSUFFICIENT_PERMISSIONS",
        message: rbacResp.val.message,
      });
    }
    await db.primary.insert(schema.keysPermissions).values(
      addPermissions.map((r) => ({
        keyId: keyId,
        permissionId: r.id,
        workspaceId: auth.authorizedWorkspaceId,
      })),
    );
  }
  c.executionCtx.waitUntil(
    Promise.all([cache.keyById.remove(key.id), cache.keyByHash.remove(key.hash)]),
  );

  c.executionCtx.waitUntil(
    analytics.ingestUnkeyAuditLogs([
      ...disconnectPermissions.map((r) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "authorization.disconnect_permission_and_key" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Disconnected ${r.permissionId} and ${key.id}`,
        resources: [
          {
            type: "permission" as const,
            id: r.permissionId,
          },
          {
            type: "key" as const,
            id: key.id,
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      })),
      ...addPermissions.map((r) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "authorization.connect_permission_and_key" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Connected ${r.id} and ${keyId}`,
        resources: [
          {
            type: "permission" as const,
            id: r.id,
          },
          {
            type: "key" as const,
            id: keyId,
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      })),
      ...createPermissions.map((r) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "permission.create" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Created ${r.id}`,
        resources: [
          {
            type: "permission" as const,
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
      ...addPermissions.map((r) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "authorization.connect_permission_and_key" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Connected ${r.id} and ${keyId}`,
        resources: [
          {
            type: "permission" as const,
            id: r.id,
          },
          {
            type: "key" as const,
            id: keyId,
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      })),
    ]),
  );
  return allPermissions;
}
