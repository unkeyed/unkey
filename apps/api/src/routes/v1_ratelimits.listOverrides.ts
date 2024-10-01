import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "listOverrides",
  method: "get",
  path: "/v1/ratelimit.listOverrides",
  security: [{ bearerAuth: [] }],
  request: {
    query: z.object({
      ratelimitId: z.string().min(1).openapi({
        description: "The id of the permission to fetch",
        example: "ratelimit_123",
      }),
    }),
  },
  responses: {
    200: {
      description: "The Overrides on your ratelimit",
      content: {
        "application/json": {
          schema: z.array(
            z.object({
              id: z.string().openapi({
                description: "The id of the ratelimit override",
                example: "ratelimit_1234",
              }),
              namespace: z.string().optional().default("default").openapi({
                description:
                  "Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
                example: "email.outbound",
              }),
              identifier: z.string().openapi({
                description:
                  "Identifier of your user, this can be their userId, an email, an ip or anything else.",
                example: "user_123",
              }),
              limit: z.number().int().positive().openapi({
                description: "How many requests may pass in a given window.",
                example: 10,
              }),
              duration: z.number().int().min(1000).openapi({
                description: "The window duration in milliseconds",
                example: 60_000,
              }),
              async: z.boolean().default(false).optional().openapi({
                description:
                  "Async will return a response immediately, lowering latency at the cost of accuracy.",
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
export type V1RatelimitListOverridesResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;
export const registerV1RatelimitListOverrides = (app: App) =>
  app.openapi(route, async (c) => {
    const { db } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.read_override")),
    );

    const overrides = await db.readonly.query.ratelimitOverrides.findMany({
      where: (table, { eq }) => eq(table.id, auth.authorizedWorkspaceId),
      with: {
        namespace: {
            columns: {
              name: true,
            },
          },
      }
    });

    return c.json(
        overrides.map((r) => ({
        id: r.id,
        namespace: r.namespace.name,
        identifier: r.identifier,
        limit: r.limit,
        duration: r.duration,
        async: r.async ?? undefined,
      })),
    );
  });
