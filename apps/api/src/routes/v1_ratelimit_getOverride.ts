import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "getOverride",
  method: "get",
  path: "/v1/ratelimit.getOverride",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
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
export const registerV1RdatelimitGetOverride = (app: App) =>
  app.openapi(route, async (c) => {
    const { identifier } = c.req.valid("query");
    const { db } = c.get("services");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.read_override")),
    );
    if (!auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "UNAUTHORIZED",
        message: "Missing required permission: ratelimit.*.read_override",
      });
    }

    const override = await db.primary.query.ratelimitOverrides.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId), eq(table.identifier, identifier)),
    });
    if (!override) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "Override not found",
      });
    }

    return c.json({
      id: override.id,
      identifier: override.identifier,
      limit: override.limit,
      duration: override.duration,
      async: override.async ?? undefined,
    });
  });
