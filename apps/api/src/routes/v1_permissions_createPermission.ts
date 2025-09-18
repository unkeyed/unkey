import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { validation } from "@unkey/validation";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { DatabaseError } from "@planetscale/database";
import { DrizzleQueryError, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "createPermission",
  summary: "Create permission",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/permissions.createPermission",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: validation.identifier.openapi({
              description: "The unique name of your permission.",
              example: "Can write records",
            }),
            slug: validation.identifier.optional().openapi({
              description: "The unique slug of your permission. If not provided, the name is used.",
              example: "record.write",
            }),
            description: validation.description.optional().openapi({
              description:
                "Explain what this permission does. This is just for your team, your users will not see this.",
              example: "record.write can create new dns records for our domains.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Sucessfully created a permission",
      content: {
        "application/json": {
          schema: z.object({
            permissionId: z.string().openapi({
              description: "The id of the permission. This is used internally",
              example: "perm_123",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1PermissionsCreatePermissionRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1PermissionsCreatePermissionResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1PermissionsCreatePermission = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_permission")),
    );

    const { db } = c.get("services");

    const permission = {
      id: newId("permission"),
      workspaceId: auth.authorizedWorkspaceId,
      name: req.name,
      slug: req.slug ?? req.name,
      description: req.description,
    };

    await db.primary.transaction(async (tx) => {
      await tx
        .insert(schema.permissions)
        .values(permission)
        .catch((e) => {
          if (
            e instanceof DrizzleQueryError &&
            e.cause instanceof DatabaseError &&
            e.cause.message.includes("Duplicate entry")
          ) {
            throw new UnkeyApiError({
              code: "CONFLICT",
              message: `Permission with name "${permission.name}" already exists in this workspace`,
            });
          }

          throw e;
        });

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: auth.authorizedWorkspaceId,
        event: "permission.create",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Created ${permission.id}`,
        resources: [
          {
            type: "permission",
            id: permission.id,
            meta: {
              name: permission.name,
              description: permission.description,
            },
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    return c.json({
      permissionId: permission.id,
    });
  });
