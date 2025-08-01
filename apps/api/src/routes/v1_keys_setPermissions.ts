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
  operationId: "setPermissions",
  summary: "Set key permissions",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
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
                z.object({
                  id: z.string().min(3).optional().openapi({
                    description:
                      "The id of the permission. Provide either `id` or `name`. If both are provided `id` is used.",
                  }),
                  name: z.string().min(1).optional().openapi({
                    deprecated: true,
                    description:
                      "This field is deprecated and will be removed in a future release. please use `slug` instead.",
                  }),
                  slug: z.string().min(1).optional().openapi({
                    description:
                      "Identify the permission via its slug. Provide either `id` or `slug`. If both are provided `id` is used.",
                  }),
                  create: z
                    .boolean()
                    .optional()
                    .openapi({
                      description: `Set to true to automatically create the permissions they do not exist yet. Only works when specifying \`name\`.
                Autocreating permissions requires your root key to have the \`rbac.*.create_permission\` permission, otherwise the request will get rejected`,
                    }),
                }),
              )
              .min(1)
              .openapi({
                description: `The permissions you want to set for this key. This overwrites all existing permissions.
            Setting permissions requires the \`rbac.*.add_permission_to_key\` permission.`,
                example: [
                  {
                    id: "perm_123",
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

    const allPermissions = await setPermissions(c, auth, req.keyId, req.permissions);

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
  requested: Array<{
    id?: string;
    name?: string;
    slug?: string;
    create?: boolean;
  }>,
): Promise<Array<{ id: string; name: string }>> {
  const { db, cache, rbac } = c.get("services");

  const requestedIds = requested.filter(({ id }) => !!id).map(({ id }) => id!);
  const requestedSlugs = requested
    .filter(({ name, slug }) => !!name || !!slug)
    .map(({ name, slug, create }) => ({ slug: slug ?? name!, create: create! }));

  const [key, existingPermissions, connectedPermissions] = await Promise.all([
    db.primary.query.keys.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          eq(table.id, keyId),
          isNull(table.deletedAtM),
        ),
    }),
    db.primary.query.permissions.findMany({
      where: (table, { eq, or, and, inArray }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          or(
            requestedIds.length > 0 ? inArray(table.id, requestedIds) : undefined,
            requestedSlugs.length > 0
              ? inArray(
                  table.slug,
                  requestedSlugs.map((r) => r.slug),
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
            slug: true,
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
    if (requestedIds.includes(r.permissionId)) {
      return false;
    }
    if (requestedSlugs.some(({ slug }) => slug === r.permission.slug)) {
      return false;
    }
    if (!r.permission.slug && !r.permission.name) {
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

  const missingPermissionSlugs: string[] = [];
  for (const id of requestedIds) {
    if (!existingPermissions.some((r) => r.id === id)) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `permission ${id} not found`,
      });
    }
  }
  for (const { create, slug } of requestedSlugs) {
    if (!existingPermissions.some((r) => r.slug === slug)) {
      if (!create) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `permission ${slug} not found and not allowed to create`,
        });
      }
      missingPermissionSlugs.push(slug);
    }
  }

  const createPermissions = missingPermissionSlugs.map((slug) => ({
    id: newId("permission"),
    workspaceId: auth.authorizedWorkspaceId,
    name: slug,
    slug: slug,
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
  const auditLogs = [
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
  ];
  await insertUnkeyAuditLog(c, undefined, auditLogs);
  return allPermissions;
}
