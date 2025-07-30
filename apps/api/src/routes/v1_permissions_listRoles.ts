import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["permissions"],
  operationId: "listRoles",
  summary: "List roles",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/permissions.listRoles",
  security: [{ bearerAuth: [] }],

  responses: {
    200: {
      description: "The Roles in your workspace",
      content: {
        "application/json": {
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the role",
                example: "role_1234",
              }),

              name: z.string().openapi({
                description: "The name of the role.",
                example: "domain.record.manager",
              }),
              description: z.string().optional().openapi({
                description:
                  "The description of what this role does. This is just for your team, your users will not see this.",
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
export type V1PermissionsListRolesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1PermissionsListRoles = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.read_role")),
    );

    const roles = await db.readonly.query.roles.findMany({
      where: (table, { eq }) => eq(table.workspaceId, auth.authorizedWorkspaceId),
    });

    return c.json(
      roles.map((r) => ({
        id: r.id,
        name: r.name,
        description: r.description ?? undefined,
      })),
    );
  });
