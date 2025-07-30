import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { eq, schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["ratelimits"],
  operationId: "deleteOverride",
  summary: "Delete rate limit override",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post",
  path: "/v1/ratelimits.deleteOverride",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            namespaceId: z.string().optional().openapi({
              description:
                "The id of the namespace. Either namespaceId or namespaceName must be provided",
              example: "rlns_1234",
            }),
            namespaceName: z.string().optional().openapi({
              description:
                "The name of the namespace. Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
              example: "email.outbound",
            }),
            identifier: z.string().openapi({
              description:
                "Identifier of your user, this can be their userId, an email, an ip or anything else. Wildcards ( * ) can be used to match multiple identifiers, More info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
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
    if (!namespaceId && !namespaceName) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "You must provide a namespaceId or a namespaceName",
      });
    }
    const { db } = c.get("services");

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;

    await db.primary.transaction(async (tx) => {
      const namespace = await db.primary.query.ratelimitNamespaces.findFirst({
        where: (table, { eq, and }) =>
          and(
            eq(table.workspaceId, authorizedWorkspaceId),
            namespaceId ? eq(table.id, namespaceId) : eq(table.name, namespaceName!),
          ),
        with: {
          overrides: {
            where: (table, { eq, and, isNull }) =>
              and(isNull(table.deletedAtM), eq(table.identifier, identifier)),
          },
        },
      });

      if (!namespace) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `Namespace ${namespaceId ? namespaceId : namespaceName} not found`,
        });
      }
      const override = namespace.overrides.at(0);

      if (!override) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `Override with ${identifier} identifier not found`,
        });
      }
      await tx
        .update(schema.ratelimitOverrides)
        .set({ deletedAtM: Date.now() })
        .where(eq(schema.ratelimitOverrides.id, override.id));

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
