import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "addPermissions",
  summary: "Add key permissions",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/keys.addPermissions",
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
                description: "The permissions you want to add to this key",
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
export type V1KeysAddPermissionsRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysAddPermissionsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysAddPermissions = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.add_permission_to_key")),
    );

    const { db, rbac, cache } = c.get("services");

    const requestedIds = req.permissions.filter((p) => "id" in p).map((p) => p.id!);
    const requestedSlugs = req.permissions.filter((p) => "name" in p).map((p) => p.name!);

    const [key, existingPermissions, connectedPermissions] = await Promise.all([
      db.primary.query.keys.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, auth.authorizedWorkspaceId),
            eq(table.id, req.keyId),
            isNull(table.deletedAtM),
          ),
      }),
      db.primary.query.permissions.findMany({
        where: (table, { eq, or, and, inArray }) =>
          and(
            eq(table.workspaceId, auth.authorizedWorkspaceId),
            or(
              requestedIds.length > 0 ? inArray(table.id, requestedIds) : undefined,
              requestedSlugs.length > 0 ? inArray(table.slug, requestedSlugs) : undefined,
            ),
          ),
      }),
      await db.primary.query.keysPermissions.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, req.keyId)),
      }),
    ]);

    if (!key) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: `key ${req.keyId} not found` });
    }

    const missingPermissionSlugs: string[] = [];
    for (const permission of req.permissions) {
      if ("id" in permission && !existingPermissions.some((ep) => ep.id === permission.id)) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `permission ${permission.id} not found`,
        });
      }
      if ("name" in permission && !existingPermissions.some((ep) => ep.slug === permission.name)) {
        if (!permission.create) {
          throw new UnkeyApiError({
            code: "NOT_FOUND",
            message: `permission ${permission.name} not found`,
          });
        }
        missingPermissionSlugs.push(permission.name!);
      }
    }

    const createPermissions = missingPermissionSlugs.map((slug) => ({
      id: newId("permission"),
      workspaceId: auth.authorizedWorkspaceId,
      name: slug,
      slug,
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
          code: "UNAUTHORIZED",
          message: rbacResp.val.message,
        });
      }

      await db.primary.insert(schema.permissions).values(createPermissions);
    }
    const allPermissions = [
      ...existingPermissions.map((p) => ({ id: p.id, slug: p.name })),
      ...createPermissions.map((p) => ({ id: p.id, slug: p.name })),
    ];

    const addPermissions = allPermissions.filter(
      (ap) => !connectedPermissions.some((cp) => cp.permissionId === ap.id),
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
          code: "UNAUTHORIZED",
          message: rbacResp.val.message,
        });
      }
      await db.primary.insert(schema.keysPermissions).values(
        addPermissions.map((p) => ({
          keyId: req.keyId,
          permissionId: p.id,
          workspaceId: auth.authorizedWorkspaceId,
        })),
      );
    }

    c.executionCtx.waitUntil(
      Promise.all([cache.keyById.remove(key.id), cache.keyByHash.remove(key.hash)]),
    );

    await insertUnkeyAuditLog(c, undefined, [
      ...createPermissions.map((p) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "permission.create" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Created ${p.id}`,
        resources: [
          {
            type: "permission" as const,
            id: p.id,
            meta: {
              name: p.name,
            },
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      })),
      ...addPermissions.map((p) => ({
        workspaceId: auth.authorizedWorkspaceId,
        event: "authorization.connect_permission_and_key" as const,
        actor: {
          type: "key" as const,
          id: auth.key.id,
        },
        description: `Connected ${p.id} and ${req.keyId}`,
        resources: [
          {
            type: "permission" as const,
            id: p.id,
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
    ]);

    return c.json(allPermissions.map((p) => ({ id: p.id, name: p.slug })));
  });
