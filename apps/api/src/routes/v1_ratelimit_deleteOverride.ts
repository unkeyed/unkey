import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { and, eq, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "deleteOverride",
  method: "post",
  path: "/v1/ratelimit.deleteOverride",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            namespaceId: z.string().optional().openapi({
              description: "The id of the namespace.",
              example: "rlns_1234",
            }),
            namespaceName: z.string().optional().openapi({
              description:
                "The name of the namespace. Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
              example: "email.outbound",
            }),
            identifier: z.string().openapi({
              description:
                "Identifier of your user, this can be their userId, an email, an ip or anything else.",
              example: "user_123",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Successfully deleted a ratelimit override",
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
export type V1RatelimitDeleteOverrideRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1RatelimitDeleteOverrideResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1RatelimitDeleteOverride = (app: App) =>
  app.openapi(route, async (c) => {
    const { namespaceId, namespaceName, identifier } = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.delete_override")),
    );
    // Todo Add an option for namespaceName to be used instead of namespaceId
    const { db } = c.get("services");
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const namespace = await db.primary.query.ratelimitNamespaces.findFirst({
      where: (table, { eq, and }) =>
        and(
          eq(table.workspaceId, authorizedWorkspaceId),
          namespaceId ? eq(table.id, namespaceId) : eq(table.name, namespaceName!),
        ),
      with: {
        overrides: {
          where: and(eq(schema.ratelimitOverrides.identifier, identifier)),
        },
      },
    });
    if (!namespace) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `Namespace ${namespaceId ? namespaceId : namespaceName} not found`,
      });
    }

    await db.primary.transaction(async (tx) => {
      const override = await tx.query.ratelimitOverrides.findFirst({
        where: and(
          eq(schema.ratelimitOverrides.workspaceId, auth.authorizedWorkspaceId),
          eq(schema.ratelimitOverrides.namespaceId, namespace.id),
          eq(schema.ratelimitOverrides.identifier, identifier),
        ),
      });

      if (!override) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `Override ${identifier} in namespace ${namespaceId} not found`,
        });
      }
      await tx.delete(schema.ratelimitOverrides);

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: auth.authorizedWorkspaceId,
        event: "ratelimit.delete_override",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Deleted ratelimit override ${override.id}`,
        resources: [
          {
            type: "ratelimitOverride",
            id: override.id,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });
    });

    return c.json({});
  });
