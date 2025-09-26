import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["identities"],
  operationId: "getIdentity",
  summary: "Get identity",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/identities.getIdentity",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      identityId: z.string().min(1).optional().openapi({
        description:
          "The id of the identity to fetch, use either `identityId` or `externalId`, if both are provided, `identityId` takes precedence.",
        example: "id_1234",
      }),
      externalId: z.string().min(1).optional().openapi({
        description:
          "The externalId of the identity to fetch, use either `identityId` or `externalId`, if both are provided, `identityId` takes precedence.",
        example: "id_1234",
      }),
    }),
  },
  responses: {
    200: {
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            id: z.string().openapi({
              description: "The id of this identity. Used to interact with unkey's API",
            }),
            externalId: z.string().openapi({
              description: "The id in your system",
            }),
            meta: z.record(z.unknown()).openapi({
              description: "The meta object defined for this identity.",
            }),
            ratelimits: z
              .array(
                z.object({
                  name: z.string().openapi({
                    description:
                      "The name of this limit. You will need to use this again when verifying a key.",
                    example: "tokens",
                  }),
                  limit: z.number().int().min(1).openapi({
                    description:
                      "How many requests may pass within a given window before requests are rejected.",
                    example: 10,
                  }),
                  duration: z.number().int().min(1000).openapi({
                    description: "The duration for each ratelimit window in milliseconds.",
                    example: 1000,
                  }),
                }),
              )
              .openapi({
                description:
                  "When verifying keys, you can specify which limits you want to use and all keys attached to this identity, will share the limits.",
              }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1IdentitiesGetIdentityResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1IdentitiesGetIdentity = (app: App) =>
  app.openapi(route, async (c) => {
    const { identityId, externalId } = c.req.valid("query");
    const { db } = c.get("services");

    if (!identityId && !externalId) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "Provide either identityId or externalId",
      });
    }

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) =>
        or("identity.*.read_identity", `identity.${identityId}.read_identity`),
      ),
    );

    const identity = await db.readonly.query.identities.findFirst({
      where: (table, { eq, and }) =>
        and(
          eq(table.workspaceId, auth.authorizedWorkspaceId),
          identityId ? eq(table.id, identityId) : eq(table.externalId, externalId!),
        ),
      with: {
        ratelimits: true,
      },
    });

    if (!identity || identity.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `identity ${identityId} not found`,
      });
    }

    return c.json({
      id: identity.id,
      externalId: identity.externalId,
      meta: identity.meta ?? {},
      ratelimits: identity.ratelimits.map((r) => ({
        name: r.name,
        limit: r.limit,
        duration: r.duration,
      })),
    });
  });
