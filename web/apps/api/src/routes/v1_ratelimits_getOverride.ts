import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["ratelimits"],
  operationId: "getOverride",
  summary: "Get rate limit override",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/ratelimits.getOverride",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      namespaceId: z.string().optional().openapi({
        description:
          "The id of the namespace. Either namespaceId or namespaceName must be provided",
        example: "rlns_1234",
      }),
      namespaceName: z.string().optional().openapi({
        description:
          "Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes. Wildcards can also be used, more info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
        example: "email.outbound",
      }),
      identifier: z.string().openapi({
        description:
          "Identifier of your user, this can be their userId, an email, an ip or anything else. Wildcards ( * ) can be used to match multiple identifiers, More info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
        example: "user_123",
      }),
    }),
  },
  responses: {
    200: {
      description: "Details of the override for the given identifier",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string(),
            identifier: z.string(),
            limit: z.number().int(),
            duration: z.number().int(),
            async: z.boolean().nullable().optional(),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;

export type V1RatelimitGetOverrideResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export type V1RatelimitGetOverrideRequest = z.infer<(typeof route.request)["query"]>;
export const registerV1RatelimitGetOverride = (app: App) =>
  app.openapi(route, async (c) => {
    const { namespaceId, namespaceName, identifier } = c.req.valid("query");
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("ratelimit.*.read_override")),
    );

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    if (!authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "Missing required permission: ratelimit.*.read_override",
      });
    }
    if (!namespaceId && !namespaceName) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "You must provide a namespaceId or a namespaceName",
      });
    }
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
      throw new UnkeyApiError({ code: "NOT_FOUND", message: "Namespace not found" });
    }

    const override = namespace.overrides.at(0);
    if (!override) {
      throw new UnkeyApiError({ code: "NOT_FOUND", message: "Override not found" });
    }
    return c.json({
      id: override.id,
      identifier: override.identifier,
      limit: override.limit,
      duration: override.duration,
      async: override.async,
    });
  });
