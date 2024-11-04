import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["ratelimit"],
  operationId: "ratelimit.setOverride",
  method: "post",
  path: "/v1/ratelimit.setOverride",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            namespaceId: z.string().openapi({
              description:
                "Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
              example: "email.outbound",
            }),
            identifier: z.string().openapi({
              description:
                "Identifier of your user, this can be their userId, an email, an ip or anything else. Wildcards ( * ) can be used to match multiple identifiers, More info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
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
              description:
                "Async will return a response immediately, lowering latency at the cost of accuracy.",
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
      buildUnkeyQuery(({ or }) =>
        or(
          "*",
          "ratelimit.*.set_override",
          "ratelimit.*.read_override",
        ),
      ),
    );
    // console.log(req);
    // Re work db call and update if needed
    const { db, analytics } = c.get("services");
    await db.primary.transaction(async (tx) => {
      const res = await tx
        .insert(schema.ratelimitOverrides)
        .values({
          id: newId("ratelimitOverride"),
          workspaceId: auth.authorizedWorkspaceId,
          createdAt: new Date(),
          namespaceId: req.namespaceId,
          identifier: req.identifier,
          limit: req.limit,
          duration: req.duration,
          async: req.async,
        })
        .onDuplicateKeyUpdate({
          set: {
            limit: req.limit,
            duration: req.duration,
            async: req.async,
          },
        });
      let auditType: "ratelimitOverride.create" | "ratelimitOverride.update" =
        "ratelimitOverride.create";
      if (!res.statement.startsWith("insert")) {
        auditType = "ratelimitOverride.update";
      }

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: auth.authorizedWorkspaceId,
        event: "ratelimit.set_override",
        actor: {
          type: "key",
          id: auth.key.id,
        },
        description: `Set ratelimit override for ${req.namespaceId} and ${req.identifier}`,
        resources: [
          {
            type: "ratelimitOverride",
            id: res.statement.startsWith("insert")
              ? res.insertId
              : JSON.stringify(res.rows[0].entries[0].id),
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });

      c.executionCtx.waitUntil(
        analytics.ingestUnkeyAuditLogsTinybird({
          workspaceId: auth.authorizedWorkspaceId,
          event: "ratelimit.set_override",
          actor: {
            type: "key",
            id: auth.key.id,
          },
          description: `Set ratelimit override for ${req.namespaceId} and ${req.identifier}`,
          resources: [
            {
              type: "ratelimitOverride",
              id: res.statement.startsWith("insert")
                ? res.insertId
                : JSON.stringify(res.rows[0].entries[0].id),
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        }),
      );
    });
    return c.json({
      success: true,
      message: "Ratelimit override has been set.",
    });
  });
