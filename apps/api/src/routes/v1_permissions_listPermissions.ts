import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["permissions"],
  operationId: "listPermissions",
  method: "get",
  path: "/v1/permissions.listPermissions",
  security: [{ bearerAuth: [] }],
  responses: {
    200: {
      description: "The permissions in your workspace",
      content: {
        "application/json": {
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the permission",
                example: "perm_123",
              }),

              name: z.string().openapi({
                description: "The name of the permission.",
                example: "domain.record.manager",
              }),
              description: z.string().optional().openapi({
                description:
                  "The description of what this permission does. This is just for your team, your users will not see this.",
                example: "Can manage dns records",
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
export type V1PermissionsListPermissionsResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1PermissionsListPermissions = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.read_permission")),
    );

    const permissions = await db.readonly.query.permissions.findMany({
      where: (table, { eq }) => eq(table.workspaceId, auth.authorizedWorkspaceId),
    });

    return c.json(
      permissions.map((p) => ({
        id: p.id,
        name: p.name,
        description: p.description ?? undefined,
      })),
    );
  });
