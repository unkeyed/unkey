import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { RatelimitNamespace, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  method: "post",
  path: "/v1/ratelimits.limit",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
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
            cost: z.number().int().min(1).default(1).optional().openapi({
              description:
                "Expensive requests may use up more tokens. You can specify a cost to the request here and we'll deduct this many tokens in the current window. If there are not enough tokens left, the request is denied.",
              example: 2,
              default: 1,
            }),
            async: z.boolean().default(false).optional().openapi({
              description:
                "Async will return a response immediately, lowering latency at the cost of accuracy.",
            }),
            meta: z
              .record(z.union([z.string(), z.boolean(), z.number(), z.null()]))
              .optional()
              .openapi({
                description: "Attach any metadata to this request",
              }),
            sharding: z.enum(["edge"]).optional().openapi({
              description: "Not implemented yet",
            }),
            resources: z
              .array(
                z.object({
                  type: z.string().openapi({
                    description: "The type of resource",
                    example: "organization",
                  }),
                  id: z.string().openapi({
                    description: "The unique identifier for the resource",
                    example: "org_123",
                  }),
                  name: z.string().optional().openapi({
                    description: "A human readable name for this resource",
                    example: "unkey",
                  }),
                  meta: z
                    .record(z.union([z.string(), z.boolean(), z.number(), z.null()]))
                    .optional()
                    .openapi({
                      description: "Attach any metadata to this resources",
                    }),
                }),
              )
              .optional()
              .openapi({
                description: "Resources that are about to be accessed by the user",
                example: [
                  {
                    type: "project",
                    id: "p_123",
                    name: "dub",
                  },
                ],
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "",
      content: {
        "application/json": {
          schema: z.object({
            success: z.boolean().openapi({
              description:
                "Returns true if the request should be processed, false if it was rejected.",
              example: true,
            }),
            limit: z.number().openapi({
              description: "How many requests are allowed within a window.",
              example: 10,
            }),
            remaining: z.number().openapi({
              description: "How many requests can still be made in the current window.",
              example: 9,
            }),
            reset: z.number().openapi({
              description: "A unix millisecond timestamp when the limits reset.",
              example: 1709804263654,
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});
export type Route = typeof route;

export type V1RatelimitLimitRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1RatelimitLimitResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1RatelimitLimit = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const { cache, db, rateLimiter, analytics, rbac } = c.get("services");

    const rootKey = await rootKeyAuth(c);

    const { val, err } = await cache.withCache(
      c,
      "ratelimitByIdentifier",
      [rootKey.authorizedWorkspaceId, req.namespace, req.identifier].join("::"),
      async () => {
        const dbRes = await db.query.ratelimitNamespaces.findFirst({
          where: (table, { eq, and }) =>
            and(
              eq(table.workspaceId, rootKey.authorizedWorkspaceId),
              eq(table.name, req.namespace),
            ),
          with: {
            overrides: {
              where: (table, { eq, and }) =>
                and(
                  eq(table.workspaceId, rootKey.authorizedWorkspaceId),
                  eq(table.identifier, req.identifier),
                ),
            },
          },
        });
        if (!dbRes) {
          const canCreateNamespace = rbac.evaluatePermissions(
            buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.create_namespace")),
            rootKey.permissions ?? [],
          );
          if (canCreateNamespace.err || !canCreateNamespace.val.valid) {
            return null;
          }
          const namespace: RatelimitNamespace = {
            id: newId("ratelimitNamespace"),
            createdAt: new Date(),
            name: req.namespace,
            deletedAt: null,
            updatedAt: null,
            workspaceId: rootKey.authorizedWorkspaceId,
          };
          await db.insert(schema.ratelimitNamespaces).values(namespace);
          await analytics.ingestUnkeyAuditLogs({
            workspaceId: rootKey.authorizedWorkspaceId,
            actor: {
              type: "key",
              id: rootKey.key.id,
            },
            event: "ratelimitNamespace.create",
            description: `Created ${namespace.id}`,
            resources: [
              {
                type: "ratelimitNamespace",
                id: namespace.id,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          });

          return {
            namespace,
          };
        }
        return {
          namespace: dbRes,
          override: dbRes.overrides.at(0),
        };
      },
    );
    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load api: ${err.message}`,
      });
    }
    if (!val || val.namespace.workspaceId !== rootKey.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `namespace ${req.namespace} not found`,
      });
    }

    const authResult = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) =>
        or("*", "ratelimit.*.limit", `ratelimit.${val.namespace.id}.limit`),
      ),
      rootKey.permissions ?? [],
    );
    if (authResult.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: authResult.err.message,
      });
    }
    if (!authResult.val.valid) {
      throw new UnkeyApiError({
        code: "INSUFFICIENT_PERMISSIONS",
        message: authResult.val.message,
      });
    }

    const { override, namespace } = val;

    const limit = override?.limit ?? req.limit;
    const duration = override?.duration ?? req.duration;
    const async = typeof override?.async !== "undefined" ? override.async : req.async;
    const sharding = override?.sharding ?? req.sharding;
    const shard =
      sharding === "edge"
        ? // @ts-ignore - this is a bug in the types
          c.req.raw?.cf?.colo
        : undefined;

    const { val: ratelimitResponse, err: ratelimitError } = await rateLimiter.limit(c, {
      identifier: [namespace.id, req.identifier, limit, duration, async].join("::"),
      interval: duration,
      limit,
      shard,
      cost: req.cost,
      async: req.async,
    });
    if (ratelimitError) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: ratelimitError.message,
      });
    }
    const remaining = Math.max(0, limit - ratelimitResponse.current);
    c.executionCtx.waitUntil(
      analytics.ingestRatelimit({
        workspaceId: rootKey.authorizedWorkspaceId,

        namespaceId: namespace.id,
        requestId: c.get("requestId"),
        identifier: req.identifier,

        time: Date.now(),
        serviceLatency: -1,
        success: ratelimitResponse.pass,
        remaining,
        config: {
          limit,
          duration,
          async: async ?? false,
          sharding: sharding ?? "",
        },
        context: {
          ipAddress: c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP") ?? "",
          userAgent: c.req.header("User-Agent") ?? "",
          // @ts-expect-error - the cf object will be there on cloudflare
          country: c.req.raw?.cf?.country ?? "",
          // @ts-expect-error - the cf object will be there on cloudflare
          continent: c.req.raw?.cf?.continent ?? "",
          // @ts-expect-error - the cf object will be there on cloudflare
          city: c.req.raw?.cf?.city ?? "",
          // @ts-expect-error - the cf object will be there on cloudflare
          colo: c.req.raw?.cf?.colo ?? "",
        },
      }),
    );

    if (req.resources && req.resources.length > 0) {
      c.executionCtx.waitUntil(
        analytics
          .ingestGenericAuditLogs({
            auditLogId: newId("auditLog"),
            workspaceId: rootKey.authorizedWorkspaceId,
            bucket: namespace.id,
            actor: {
              type: "key",
              id: rootKey.key.id,
            },
            description: "ratelimit",
            event: ratelimitResponse.pass ? "ratelimit.success" : "ratelimit.denied",
            meta: {
              requestId: c.get("requestId"),
              namespacId: namespace.id,
              identifier: req.identifier,
              success: ratelimitResponse.pass,
            },
            time: Date.now(),
            resources: req.resources,
            context: {
              location: c.req.header("True-Client-IP") ?? "",
              userAgent: c.req.header("User-Agent") ?? "",
            },
          })
          .then((res) => {
            console.log(JSON.stringify({ req, res }, null, 2));
          }),
      );
    }

    return c.json({
      limit,
      remaining,
      reset: ratelimitResponse.reset,
      success: ratelimitResponse.pass,
    });
  });
