import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { eq, schema } from "@unkey/db";
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
            identifier: z.string().min(3).openapi({
              description:
                "Identifier of your user, this can be their userId, an email, an ip or anything else. Wildcards ( * ) can be used to match multiple identifiers, More info can be found at https://www.unkey.com/docs/ratelimiting/overrides#wildcard-rules",
              example: "user_123",
            }),
            limit: z.number().int().nonnegative().openapi({
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
            overrideId: z.string().openapi({
              description: "The id of the override. This is used internally",
              example: "over_123",
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
    if (!req.namespaceId && !req.namespaceName) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "You must provide a namespaceId or a namespaceName",
      });
    }
    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "ratelimit.*.set_override")),
    );

    const { db } = c.get("services");
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;

    const overrideId = await db.primary.transaction(async (tx) => {
      const namespace = await tx.query.ratelimitNamespaces.findFirst({
        where: (table, { and, eq }) =>
          and(
            eq(table.workspaceId, authorizedWorkspaceId),
            req.namespaceId ? eq(table.id, req.namespaceId) : eq(table.name, req.namespaceName!),
          ),
        with: {
          overrides: {
            where: (table, { eq }) => eq(table.identifier, req.identifier),
          },
        },
      });

      if (!namespace) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `Namespace ${req.namespaceId ? req.namespaceId : req.namespaceName} not found`,
        });
      }

      const override = namespace.overrides.at(0);
      const overrideId = override?.id ?? newId("ratelimitOverride");
      if (override) {
        await tx
          .update(schema.ratelimitOverrides)
          .set({
            limit: req.limit,
            duration: req.duration,
            async: req.async,
            updatedAtM: Date.now(),
          })
          .where(eq(schema.ratelimitOverrides.id, override.id));

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
              id: override.id,
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        });
      } else {
        await tx.insert(schema.ratelimitOverrides).values({
          id: overrideId,
          workspaceId: auth.authorizedWorkspaceId,
          createdAtM: Date.now(),
          namespaceId: namespace.id,
          identifier: req.identifier,
          limit: req.limit,
          duration: req.duration,
          async: req.async,
        });

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
              id: overrideId,
            },
          ],

          context: { location: c.get("location"), userAgent: c.get("userAgent") },
        });
      }
      return overrideId;
    });
    return c.json({
      overrideId,
    });
  });
