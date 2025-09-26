import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { DatabaseError } from "@planetscale/database";
import { DrizzleQueryError, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";
import { validation } from "@unkey/validation";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "createRole",
  summary: "Create role",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/permissions.createRole",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: validation.name.openapi({
              description: "The unique name of your role.",
              example: "dns.records.manager",
            }),
            description: validation.description.optional().openapi({
              description:
                "Explain what this role does. This is just for your team, your users will not see this.",
              example: "dns.records.manager can read and write dns records for our domains.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Sucessfully created a role",
      content: {
        "application/json": {
          schema: z.object({
            roleId: z.string().openapi({
              description: "The id of the role. This is used internally",
              example: "role_123",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1PermissionsCreateRoleRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1PermissionsCreateRoleResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1PermissionsCreateRole = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_role")),
    );

    const { db } = c.get("services");

    const role = {
      id: newId("role"),
      workspaceId: auth.authorizedWorkspaceId,
      name: req.name,
      description: req.description,
    };

    await db.primary.transaction(async (tx) => {
      await tx
        .insert(schema.roles)
        .values(role)
        .catch((e) => {
          if (
            e instanceof DrizzleQueryError &&
            e.cause instanceof DatabaseError &&
            e.cause.message.includes("Duplicate entry")
          ) {
            throw new UnkeyApiError({
              code: "CONFLICT",
              message: `Role with name "${role.name}" already exists in this workspace`,
            });
          }

          throw e;
        });

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: auth.authorizedWorkspaceId,
        event: "role.create",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Created ${role.id}`,
        resources: [
          {
            type: "role",
            id: role.id,
            meta: {
              name: role.name,
              description: role.description,
            },
          },
        ],
        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    return c.json({
      roleId: role.id,
    });
  });
