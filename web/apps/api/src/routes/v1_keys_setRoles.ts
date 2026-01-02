import type { App, Context } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, inArray, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "setRoles",
  summary: "Set key roles",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/keys.setRoles",
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
                z.object({
                  id: z.string().min(3).optional().openapi({
                    description:
                      "The id of the role. Provide either `id` or `name`. If both are provided `id` is used.",
                  }),
                  name: z.string().min(1).optional().openapi({
                    description:
                      "Identify the role via its name. Provide either `id` or `name`. If both are provided `id` is used.",
                  }),
                  create: z
                    .boolean()
                    .optional()
                    .openapi({
                      description: `Set to true to automatically create the permissions they do not exist yet. Only works when specifying \`name\`.
                Autocreating roles requires your root key to have the \`rbac.*.create_role\` permission, otherwise the request will get rejected`,
                    }),
                }),
              )
              .min(1)
              .openapi({
                description: `The roles you want to set for this key. This overwrites all existing roles.
            Setting roles requires the \`rbac.*.add_role_to_key\` permission.`,
                example: [
                  {
                    id: "role_123",
                  },
                  {
                    name: "dns.record.create",
                  },
                  {
                    name: "dns.record.delete",
                    create: true,
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
export type V1KeysSetRolesRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysSetRolesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysSetRoles = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(c);

    const allRoles = await setRoles(c, auth, req.keyId, req.roles);

    return c.json(allRoles);
  });

export async function setRoles(
  c: Context,
  auth: {
    authorizedWorkspaceId: string;
    permissions: string[];
    key: { id: string };
  },
  keyId: string,
  requested: Array<{ id?: string; name?: string; create?: boolean }>,
): Promise<Array<{ id: string; name: string }>> {
  const { db, cache, rbac } = c.get("services");

  const requestedIds = requested.filter(({ id }) => !!id).map(({ id }) => id!);
  const requestedNames = requested
    .filter(({ name }) => !!name)
    .map(({ name, create }) => ({ name: name!, create: create! }));

  const [key, existingRoles, connectedRoles] = await Promise.all([
    db.primary.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          eq(table.id, keyId),
          isNull(table.deletedAtM),
        ),
    }),
    db.primary.query.roles.findMany({
      where: (table, { eq, or, and, inArray }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          or(
            requestedIds.length > 0 ? inArray(table.id, requestedIds) : undefined,
            requestedNames.length > 0
              ? inArray(
                  table.name,
                  requestedNames.map((n) => n.name),
                )
              : undefined,
          ),
        ),
    }),
    await db.primary.query.keysRoles.findMany({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, keyId)),
      with: {
        role: {
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

  const disconnectRoles = connectedRoles.filter((r) => {
    if (requestedIds.includes(r.roleId)) {
      return false;
    }
    if (requestedNames.some(({ name }) => name === r.role.name)) {
      return false;
    }
    return true;
  });

  if (disconnectRoles.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.remove_role_from_key")),
      auth.permissions,
    );
    if (rbacResp.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "unable to evaluate roles",
      });
    }
    if (!rbacResp.val.valid) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: rbacResp.val.message,
      });
    }

    await db.primary.delete(schema.keysRoles).where(
      and(
        eq(schema.keysRoles.workspaceId, auth.authorizedWorkspaceId),
        eq(schema.keysRoles.keyId, key.id),
        inArray(
          schema.keysRoles.roleId,
          disconnectRoles.map((r) => r.roleId),
        ),
      ),
    );
  }

  const missingRoleNames: string[] = [];
  for (const id of requestedIds) {
    if (!existingRoles.some((r) => r.id === id)) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `role ${id} not found`,
      });
    }
  }
  for (const { create, name } of requestedNames) {
    if (!existingRoles.some((r) => r.name === name)) {
      if (!create) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `role ${name} not found and not allowed to create`,
        });
      }
      missingRoleNames.push(name);
    }
  }

  const createRoles = missingRoleNames.map((name) => ({
    id: newId("role"),
    workspaceId: auth.authorizedWorkspaceId,
    name,
  }));
  if (createRoles.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_role")),
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

    await db.primary.insert(schema.roles).values(createRoles);
  }
  const allRoles = [
    ...existingRoles.map((p) => ({ id: p.id, name: p.name })),
    ...createRoles.map((p) => ({ id: p.id, name: p.name })),
  ];

  const addRoles = allRoles.filter((ar) => !connectedRoles.some((cp) => cp.roleId === ar.id));
  if (addRoles.length > 0) {
    const rbacResp = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.add_role_to_key")),
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
    await db.primary.insert(schema.keysRoles).values(
      addRoles.map((r) => ({
        keyId: keyId,
        roleId: r.id,
        workspaceId: auth.authorizedWorkspaceId,
      })),
    );
  }
  c.executionCtx.waitUntil(
    Promise.all([cache.keyById.remove(key.id), cache.keyByHash.remove(key.hash)]),
  );

  const auditLogs = [
    ...disconnectRoles.map((r) => ({
      workspaceId: auth.authorizedWorkspaceId,
      event: "authorization.disconnect_role_and_key" as const,
      actor: {
        type: "key" as const,
        id: auth.key.id,
      },
      description: `Disconnected ${r.roleId} and ${key.id}`,
      resources: [
        {
          type: "role" as const,
          id: r.roleId,
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
    ...addRoles.map((r) => ({
      workspaceId: auth.authorizedWorkspaceId,
      event: "authorization.connect_role_and_key" as const,
      actor: {
        type: "key" as const,
        id: auth.key.id,
      },
      description: `Connected ${r.id} and ${keyId}`,
      resources: [
        {
          type: "role" as const,
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
      description: `Connected ${r.id} and ${keyId}`,
      resources: [
        {
          type: "role" as const,
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
  ];
  await insertUnkeyAuditLog(c, undefined, auditLogs);

  return allRoles;
}
