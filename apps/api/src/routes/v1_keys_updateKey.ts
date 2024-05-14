import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";
import { eq } from "drizzle-orm";

const route = createRoute({
  tags: ["keys"],
  operationId: "updateKey",
  method: "post",
  path: "/v1/keys.updateKey",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().openapi({
              description: "The id of the key you want to modify",
              example: "key_123",
            }),
            name: z.string().nullish().openapi({
              description: "The name of the key",
              example: "Customer X",
            }),
            ownerId: z.string().nullish().openapi({
              description:
                "The id of the tenant associated with this key. Use whatever reference you have in your system to identify the tenant. When verifying the key, we will send this field back to you, so you know who is accessing your API.",
              example: "user_123",
            }),
            meta: z
              .record(z.unknown())
              .nullish()
              .openapi({
                description: "Any additional metadata you want to store with the key",
                example: {
                  roles: ["admin", "user"],
                  stripeCustomerId: "cus_1234",
                },
              }),
            expires: z.number().nullish().openapi({
              description:
                "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
              example: Date.now(),
            }),
            ratelimit: z
              .object({
                type: z.enum(["fast", "consistent"]).openapi({
                  description:
                    "Fast ratelimiting doesn't add latency, while consistent ratelimiting is more accurate.",
                  externalDocs: {
                    description: "Learn more",
                    url: "https://unkey.dev/docs/features/ratelimiting",
                  },
                }),
                limit: z.number().int().min(1).openapi({
                  description: "The total amount of burstable requests.",
                }),
                refillRate: z.number().int().min(1).openapi({
                  description: "How many tokens to refill during each refillInterval.",
                }),
                refillInterval: z.number().int().min(1).openapi({
                  description:
                    "Determines the speed at which tokens are refilled, in milliseconds.",
                }),
              })
              .nullish()
              .openapi({
                description:
                  "Unkey comes with per-key ratelimiting out of the box. Set `null` to disable.",
                example: {
                  type: "fast",
                  limit: 10,
                  refillRate: 1,
                  refillInterval: 60,
                },
              }),
            remaining: z.number().nullish().openapi({
              description:
                "The number of requests that can be made with this key before it becomes invalid. Set `null` to disable.",
              example: 1000,
            }),
            refill: z
              .object({
                interval: z.enum(["daily", "monthly"]).openapi({
                  description:
                    "Unkey will automatically refill verifications at the set interval. If null is used the refill functionality will be removed from the key.",
                }),
                amount: z.number().int().min(1).openapi({
                  description:
                    "The amount of verifications to refill for each occurrence is determined individually for each key.",
                }),
              })
              .nullable()
              .optional()
              .openapi({
                description:
                  "Unkey enables you to refill verifications for each key at regular intervals.",
                example: {
                  interval: "daily",
                  amount: 100,
                },
              }),
            enabled: z.boolean().optional().openapi({
              description:
                "Set if key is enabled or disabled. If disabled, the key cannot be used to verify.",
              example: true,
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description:
        "The key was successfully updated, it may take up to 30s for this to take effect in all regions",
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
export type V1KeysUpdateKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysUpdateKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysUpdate = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const { cache, db, usageLimiter, analytics } = c.get("services");

    await db.primary.transaction(async (tx) => {
      const key = await tx.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, req.keyId),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      });

      if (!key) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `key ${req.keyId} not found`,
        });
      }
      const auth = await rootKeyAuth(
        c,
        buildUnkeyQuery(({ or }) =>
          or("*", "api.*.update_key", `api.${key.keyAuth.api.id}.update_key`),
        ),
      );
      if (key.workspaceId !== auth.authorizedWorkspaceId) {
        throw new UnkeyApiError({
          code: "NOT_FOUND",
          message: `key ${req.keyId} not found`,
        });
      }

      if (req.remaining === null && req.refill) {
        throw new UnkeyApiError({
          code: "BAD_REQUEST",
          message: "Cannot set refill on a key with unlimited requests",
        });
      }
      if (req.refill && key.remaining === null) {
        throw new UnkeyApiError({
          code: "BAD_REQUEST",
          message: "Cannot set refill on a key with unlimited requests",
        });
      }

      const authorizedWorkspaceId = auth.authorizedWorkspaceId;
      const rootKeyId = auth.key.id;

      await tx
        .update(schema.keys)
        .set({
          name: req.name,
          ownerId: req.ownerId,
          meta: typeof req.meta === "undefined" ? undefined : JSON.stringify(req.meta ?? {}),
          expires:
            typeof req.expires === "undefined"
              ? undefined
              : req.expires === null
                ? null
                : new Date(req.expires),
          remaining: req.remaining,
          ratelimitType: req.ratelimit === null ? null : req.ratelimit?.type,
          ratelimitLimit: req.ratelimit === null ? null : req.ratelimit?.limit,
          ratelimitRefillRate: req.ratelimit === null ? null : req.ratelimit?.refillRate,
          ratelimitRefillInterval: req.ratelimit === null ? null : req.ratelimit?.refillInterval,
          refillInterval: req.refill === null ? null : req.refill?.interval,
          refillAmount: req.refill === null ? null : req.refill?.amount,
          lastRefillAt: req.refill == null || req.refill?.amount == null ? null : new Date(),
          enabled: req.enabled,
        })
        .where(eq(schema.keys.id, req.keyId));

      await analytics.ingestUnkeyAuditLogs({
        workspaceId: authorizedWorkspaceId,
        event: "key.update",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `Updated key config: ${Object.entries(req)
          .filter(([_, v]) => v !== undefined)
          .reduce((description, [k, v]) => {
            return `${description}${description.length > 0 ? ", " : ""}${k}=${v}`;
          }, "")}`,
        resources: [
          {
            type: "key",
            id: key.id,
          },
        ],

        context: { location: c.get("location"), userAgent: c.get("userAgent") },
      });

      await Promise.all([
        usageLimiter.revalidate({ keyId: key.id }),
        cache.keyByHash.remove(key.hash),
        cache.keyById.remove(key.id),
      ]);
    });

    return c.json({});
  });
