import type { App, Context } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertGenericAuditLogs, insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { match } from "@/pkg/util/wildcard";
import { DatabaseError } from "@planetscale/database";
import { type InsertRatelimitNamespace, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimits"],
  operationId: "limit",
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
            cost: z
              .number()
              .int()
              .min(0)
              .default(1)
              .optional()
              .openapi({
                description: `Expensive requests may use up more tokens. You can specify a cost to the request here and we'll deduct this many tokens in the current window.
If there are not enough tokens left, the request is denied.

Set it to 0 to receive the current limit without changing anything.`,
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
            // sharding: z.enum(["edge"]).optional().openapi({
            //   description: "Not implemented yet",
            // }),
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
            limit: z.number().int().openapi({
              description: "How many requests are allowed within a window.",
              example: 10,
            }),
            remaining: z.number().int().openapi({
              description: "How many requests can still be made in the current window.",
              example: 9,
            }),
            reset: z.number().int().openapi({
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
    const { cache, db, rateLimiter, analytics, rbac, logger } = c.get("services");

    const rootKey = await rootKeyAuth(c);

    const { val, err } = await cache.ratelimitByIdentifier.swr(
      [rootKey.authorizedWorkspaceId, req.namespace, req.identifier].join("::"),
      async () => {
        const dbRes = await db.readonly.query.ratelimitNamespaces.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(
              eq(table.workspaceId, rootKey.authorizedWorkspaceId),
              eq(table.name, req.namespace),
              isNull(table.deletedAtM),
            ),
          columns: {
            id: true,
            workspaceId: true,
          },
          with: {
            overrides: {
              columns: {
                identifier: true,
                async: true,
                limit: true,
                duration: true,
                sharding: true,
              },
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
          let namespace: InsertRatelimitNamespace = {
            id: newId("ratelimitNamespace"),
            createdAtM: Date.now(),
            name: req.namespace,
            deletedAtM: null,
            updatedAtM: null,
            workspaceId: rootKey.authorizedWorkspaceId,
          };
          try {
            await db.primary.insert(schema.ratelimitNamespaces).values(namespace);
            await insertUnkeyAuditLog(c, undefined, {
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
          } catch (e) {
            if (e instanceof DatabaseError && e.body.message.includes("desc = Duplicate entry")) {
              /**
               * Looks like it exists already, so let's load it
               */
              namespace = (await db.readonly.query.ratelimitNamespaces.findFirst({
                where: (table, { eq, and }) =>
                  and(
                    eq(table.name, req.namespace),
                    eq(table.workspaceId, rootKey.authorizedWorkspaceId),
                  ),
              }))!;
            } else {
              throw e;
            }
          }

          return {
            namespace,
          };
        }

        const exactMatch = dbRes.overrides.find((o) => o.identifier === req.identifier);
        if (exactMatch) {
          return {
            namespace: dbRes,
            override: exactMatch,
          };
        }
        const wildcardMatch = dbRes.overrides.find((o) => {
          if (!o.identifier.includes("*")) {
            return false;
          }
          return match(o.identifier, req.identifier);
        });
        if (wildcardMatch) {
          return {
            namespace: dbRes,
            override: wildcardMatch,
          };
        }

        return {
          namespace: dbRes,
          override: undefined,
        };
      },
    );
    if (err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to load ratelimit: ${err.message}`,
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
    const sharding = override?.sharding; //?? req.sharding;
    const shard =
      sharding === "edge"
        ? // @ts-ignore - this is a bug in the types
          c.req.raw?.cf?.colo
        : "global";
    const { val: ratelimitResponse, err: ratelimitError } = await rateLimiter.limit(c, {
      name: "default",
      workspaceId: rootKey.authorizedWorkspaceId,
      namespaceId: namespace.id,
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
      analytics
        .insertRatelimit({
          workspace_id: rootKey.authorizedWorkspaceId,
          namespace_id: namespace.id,
          request_id: c.get("requestId"),
          identifier: req.identifier,
          time: Date.now(),
          passed: ratelimitResponse.passed,
        })
        .then(({ err }) => {
          if (err) {
            logger.error("inserting ratelimit event failed", {
              error: err.message,
            });
          }
        }),
    );

    if (req.resources && req.resources.length > 0) {
      c.executionCtx.waitUntil(
        insertGenericAuditLogs(c, undefined, {
          auditLogId: newId("auditLog"),
          workspaceId: rootKey.authorizedWorkspaceId,
          bucket: namespace.id,
          actor: {
            type: "key",
            id: rootKey.key.id,
          },
          description: "ratelimit",
          event: ratelimitResponse.passed ? "ratelimit.success" : "ratelimit.denied",
          meta: {
            requestId: c.get("requestId"),
            namespacId: namespace.id,
            identifier: req.identifier,
            success: ratelimitResponse.passed,
          },
          time: Date.now(),
          resources: req.resources ?? [],
          context: {
            location: c.req.header("True-Client-IP") ?? "",
            userAgent: c.req.header("User-Agent") ?? "",
          },
        }),
      );
    }

    const res = {
      limit,
      remaining,
      reset: ratelimitResponse.reset,
      success: ratelimitResponse.passed,
    };

    c.executionCtx.waitUntil(replayToAws(c, req, res).catch(() => {}));
    return c.json(res);
  });

async function replayToAws(
  c: Context,
  req: V1RatelimitLimitRequest,
  res: V1RatelimitLimitResponse,
): Promise<void> {
  const { metrics, logger } = c.get("services");

  logger.info("replaying to aws");
  const t0 = performance.now();
  const resp = await fetch("https://api.unkey.com/v2/ratelimit.limit", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: c.req.header("authorization") ?? "",
      "X-Unkey-Metrics": "disabled",
    },
    body: JSON.stringify({
      namespace: req.namespace,
      identifier: req.identifier,
      limit: req.limit,
      duration: req.duration,
      cost: req.cost,
    }),
  });
  const awsLatency = performance.now() - t0;

  const body = await resp.json<{ data: { success: boolean } }>();

  logger.info("aws response", {
    status: resp.status,
    body: body,
  });

  metrics.emit({
    metric: "metric.ratelimit.aws",
    awsLatency,
    awsPassed: body.data.success,
    cfPassed: res.success,
  });
}
