import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, inArray, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "removeRoles",
  method: "post",
  path: "/v1/keys.removeRoles",
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
                  }),
                ]),
              )
              .min(1)
              .openapi({
                description: "The roles you want to remove from this key",
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Success",
      content: {
        "application/json": {
          schema: z.object({}),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysRemoveRolesRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysRemoveRolesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysRemoveRoles = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "rbac.*.remove_role_from_key")),
    );

    const { db, analytics } = c.get("services");

    const [key, connectedRoles] = await Promise.all([
      db.primary.query.keys.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.id, req.keyId)),
      }),

      await db.primary.query.keysRoles.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.keyId, req.keyId)),
        with: {
          role: true,
        },
      }),
    ]);
    if (!key) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${req.keyId} not found`,
      });
    }

    const deleteRoles = connectedRoles.filter((cr) => {
      for (const deleteRequest of req.roles) {
        if ("id" in deleteRequest) {
          return cr.roleId === deleteRequest.id;
        }
        if ("name" in deleteRequest) {
          return cr.role.name === deleteRequest.name;
        }
      }
    });

    if (deleteRoles.length === 0) {
      // We have nothing to do
      return c.json({});
    }
    await db.primary.delete(schema.keysRoles).where(
      and(
        eq(schema.keysRoles.workspaceId, auth.authorizedWorkspaceId),
        eq(schema.keysRoles.keyId, key.id),
        inArray(
          schema.keysRoles.roleId,
          deleteRoles.map((r) => r.roleId),
        ),
      ),
    );

    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs(
        deleteRoles.map((r) => ({
          workspaceId: auth.authorizedWorkspaceId,
          event: "authorization.disconnect_role_and_key" as const,
          actor: {
            type: "key" as const,
            id: auth.key.id,
          },
          description: `Disonnected ${r.roleId} and ${req.keyId}`,
          resources: [
            {
              type: "role" as const,
              id: r.roleId,
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
      ),
    );

    return c.json({});
  });
