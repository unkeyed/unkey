import type { App, Context } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { and, eq, inArray, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "setOverride",
  method: "post",
  path: "/v1/ratelimits.setOverride",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            namespaceId: z.string().optional().default("default").openapi({
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
            async: z.boolean().optional().default(false).openapi({
              description: "Async will return a response immediately, lowering latency at the cost of accuracy.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "Sucessfully created a ratelimit override",
      content: {
        "application/json": {
          schema: z.object({
            success: z.boolean().openapi({
              description:
                "Returns true if the request should be processed, false if it was rejected.",
              example: true,
            }),
            message: z.string().openapi({
              description: "Message about the success of the operation.",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1RatelimitSetOverrideRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1RatelimitSetOverrideResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1RatelimitSetOverride = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.set_override")),
    );

    const { cache, logger, db, rateLimiter, analytics, rbac } = c.get("services");

    const found = await db.readonly.query.ratelimitOverrides.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, auth.authorizedWorkspaceId),
          eq(table.namespaceId, req.namespaceId),
          eq(table.identifier, req.identifier)),
    })
    if (found) {
      const override = {
        id: found.id,
        workspaceId: auth.authorizedWorkspaceId,
        namespaceId: req.namespaceId,
        identifier: req.identifier,
        limit: req.limit,
        duration: req.duration,
        async: req.async,
      };

      await db.primary.transaction(async (tx) => {
        await tx
          .update(schema.ratelimitOverrides).set(override);
        await insertUnkeyAuditLog(c, tx, {
          workspaceId: auth.authorizedWorkspaceId,
          event: "ratelimitOverride.update",
          actor: {
            type: "key",
            id: auth.key.id,
          },
          description: `Update ${override.id}`,
          resources: [
            {
              type: "ratelimit",
              id: override.id,
              meta: {
                namespaceId: override.namespaceId,
                identifier: override.identifier,
                imit: override.limit,
                duration: override.duration,
                async: override.async,
              },
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        })
      });
      c.executionCtx.waitUntil(
        analytics.ingestUnkeyAuditLogsTinybird({
          workspaceId: auth.authorizedWorkspaceId,
          event: "ratelimitOverride.create",
          actor: {
            type: "key",
            id: auth.key.id,
          },
          description: `Created ${override.id}`,
          resources: [
            {
              type: "ratelimit",
              id: override.id,
              meta: {
                name: override.namespaceId,
                identifier: override.identifier,
                imit: override.limit,
                duration: override.duration,
                async: override.async,
              },
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        }),
      );
      return c.json({
        success: true,
        message: "Ratelimit override was updated succesfuly."
      });
    } else {
      await db.primary.transaction(async (tx) => {
        const override = {
          id: newId("ratelimitOverride"),
          workspaceId: auth.authorizedWorkspaceId,
          namespaceId: req.namespaceId,
          identifier: req.identifier,
          limit: req.limit,
          duration: req.duration,
          async: req.async,
        };
        await tx
          .insert(schema.ratelimitOverrides)
          .values(override);

        await insertUnkeyAuditLog(c, tx, {
          workspaceId: auth.authorizedWorkspaceId,
          event: "ratelimitOverride.create",
          actor: {
            type: "key",
            id: auth.key.id,
          },
          description: `Created ${override.id}`,
          resources: [
            {
              type: "ratelimit",
              id: override.id,
              meta: {
                identifier: override.identifier,
                limit: override.limit,
                duration: override.duration,
                async: override.async,
              },
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        });

        c.executionCtx.waitUntil(
          analytics.ingestUnkeyAuditLogsTinybird({
            workspaceId: auth.authorizedWorkspaceId,
            event: "ratelimitOverride.create",
            actor: {
              type: "key",
              id: auth.key.id,
            },
            description: `Created ${override.id}`,
            resources: [
              {
                type: "ratelimit",
                id: override.id,
                meta: {
                  name: override.namespaceId,
                  identifier: override.identifier,
                  imit: override.limit,
                  duration: override.duration,
                  async: override.async,
                },
              },
            ],

            context: { location: c.get("location"), userAgent: c.get("userAgent") },
          }),
        );
      });

      return c.json({
        success: true,
        message: "Ratelimit override was added succesfuly."
      });
    }
  });
    
