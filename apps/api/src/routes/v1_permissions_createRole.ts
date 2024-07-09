import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["permissions"],
  operationId: "createRole",
  method: "post",
  path: "/v1/permissions.createRole",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: z
              .string()
              .min(3)
              .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
                message:
                  "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
              })
              .openapi({
                description: "The unique name of your role.",
                example: "dns.records.manager",
              }),
            description: z.string().optional().openapi({
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

    const { db, analytics } = c.get("services");

    const role = {
      id: newId("test"),
      workspaceId: auth.authorizedWorkspaceId,
      name: req.name,
      description: req.description,
    };
    await db.primary.insert(schema.roles).values(role);

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs({
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
      }),
    );

    return c.json({
      roleId: role.id,
    });
  });
