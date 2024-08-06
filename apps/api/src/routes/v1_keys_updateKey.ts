import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { eq } from "@unkey/db";
import { buildUnkeyQuery } from "@unkey/rbac";
import { setPermissions } from "./v1_keys_setPermissions";
import { setRoles } from "./v1_keys_setRoles";

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
          schema: z
            .object({
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
              expires: z.number().int().nullish().openapi({
                description:
                  "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
                example: Date.now(),
              }),
              ratelimit: z
                .object({
                  type: z
                    .enum(["fast", "consistent"])
                    .optional()
                    .openapi({
                      deprecated: true,
                      description: `Fast ratelimiting doesn't add latency, while consistent ratelimiting is more accurate.
Deprecated, use 'async' instead`,
                      externalDocs: {
                        description: "Learn more",
                        url: "https://unkey.dev/docs/features/ratelimiting",
                      },
                    }),
                  async: z
                    .boolean()
                    .default(false)
                    .optional()
                    .openapi({
                      description:
                        "Asnyc ratelimiting doesn't add latency, while sync ratelimiting is slightly more accurate.",
                      externalDocs: {
                        description: "Learn more",
                        url: "https://unkey.dev/docs/features/ratelimiting",
                      },
                    }),
                  limit: z.number().int().min(1).openapi({
                    description: "The total amount of requests allowed in a single window.",
                  }),

                  refillRate: z
                    .number()
                    .int()
                    .min(1)
                    .optional()
                    .openapi({
                      description: `How many tokens to refill during each refillInterval.
Deprecated, use 'limit' instead.`,
                      deprecated: true,
                    }),
                  refillInterval: z
                    .number()
                    .int()
                    .min(1)
                    .optional()
                    .openapi({
                      description: `Determines the speed at which tokens are refilled, in milliseconds.
Deprecated, use 'duration'`,
                      deprecated: true,
                    }),
                  duration: z
                    .number()
                    .int()
                    .min(1)
                    .optional()
                    .openapi({
                      description: `The duration of each ratelimit window, in milliseconds.
This field will become required in a future version.`,
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
              remaining: z.number().int().nullish().openapi({
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
              roles: z
                .array(
                  z.object({
                    id: z.string().min(3).optional().openapi({
                      description:
                        "The id of the role. Provide either `id` or `name`. If both are provided `id` is used.",
                    }),
                    name: z.string().min(1).optional().openapi({
                      description:
                        "Identify the role via its name. Provide either `id` or `name`. If both are provided `id` is used.",
                    }),
                    create: z
                      .boolean()
                      .optional()
                      .openapi({
                        description: `Set to true to automatically create the permissions they do not exist yet. Only works when specifying \`name\`.
                    Autocreating roles requires your root key to have the \`rbac.*.create_role\` permission, otherwise the request will get rejected`,
                      }),
                  }),
                )
                .min(1)
                .optional()
                .openapi({
                  description: `The roles you want to set for this key. This overwrites all existing roles.
                Setting roles requires the \`rbac.*.add_role_to_key\` permission.`,
                  example: [
                    {
                      id: "perm_123",
                    },
                    {
                      name: "dns.record.create",
                    },
                    {
                      name: "dns.record.delete",
                      create: true,
                    },
                  ],
                }),
              permissions: z
                .array(
                  z.object({
                    id: z.string().min(3).optional().openapi({
                      description:
                        "The id of the permission. Provide either `id` or `name`. If both are provided `id` is used.",
                    }),
                    name: z.string().min(1).optional().openapi({
                      description:
                        "Identify the permission via its name. Provide either `id` or `name`. If both are provided `id` is used.",
                    }),
                    create: z
                      .boolean()
                      .optional()
                      .openapi({
                        description: `Set to true to automatically create the permissions they do not exist yet. Only works when specifying \`name\`.
                    Autocreating permissions requires your root key to have the \`rbac.*.create_permission\` permission, otherwise the request will get rejected`,
                      }),
                  }),
                )
                .min(1)
                .optional()
                .openapi({
                  description: `The permissions you want to set for this key. This overwrites all existing permissions.
                Setting permissions requires the \`rbac.*.add_permission_to_key\` permission.`,
                  example: [
                    {
                      id: "perm_123",
                    },
                    {
                      name: "dns.record.create",
                    },
                    {
                      name: "dns.record.delete",
                      create: true,
                    },
                  ],
                }),
            })
            .openapi({
              description: `Update a key's configuration.
            The \`apis.<API_ID>.update_key\` permission is required.`,
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
    const { cache, db, usageLimiter, analytics, rbac } = c.get("services");

    const auth = await rootKeyAuth(c);

    const key = await db.primary.query.keys.findFirst({
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

    if (key.workspaceId !== auth.authorizedWorkspaceId) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: `key ${req.keyId} not found`,
      });
    }

    const rbacRes = rbac.evaluatePermissions(
      buildUnkeyQuery(({ or }) =>
        or("*", "api.*.update_key", `api.${key.keyAuth.api.id}.update_key`),
      ),
      auth.permissions,
    );
    if (rbacRes.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "unable to evaluate permissions",
      });
    }
    if (!rbacRes.val.valid) {
      throw new UnkeyApiError({
        code: "INSUFFICIENT_PERMISSIONS",
        message: rbacRes.val.message,
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

    await db.primary
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
        ratelimitAsync:
          req.ratelimit === null
            ? null
            : typeof req.ratelimit === "undefined"
              ? undefined
              : typeof req.ratelimit.async === "boolean"
                ? req.ratelimit.async
                : req.ratelimit?.type === "fast",
        ratelimitLimit:
          req.ratelimit === null ? null : req.ratelimit?.limit ? req.ratelimit?.refillRate : null,
        ratelimitDuration:
          req.ratelimit === null
            ? null
            : req.ratelimit?.duration ?? req.ratelimit?.refillInterval ?? null,
        refillInterval: req.refill === null ? null : req.refill?.interval,
        refillAmount: req.refill === null ? null : req.refill?.amount,
        lastRefillAt: req.refill == null || req.refill?.amount == null ? null : new Date(),
        enabled: req.enabled,
      })
      .where(eq(schema.keys.id, key.id));

    c.executionCtx.waitUntil(usageLimiter.revalidate({ keyId: key.id }));
    c.executionCtx.waitUntil(cache.keyByHash.remove(key.hash));
    c.executionCtx.waitUntil(cache.keyById.remove(key.id));
    c.executionCtx.waitUntil(
      analytics.ingestUnkeyAuditLogs({
        workspaceId: authorizedWorkspaceId,
        event: "key.update",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `Updated key ${key.id}`,
        resources: [
          {
            type: "key",
            id: key.id,
            meta: Object.entries(req)
              .filter(([_key, value]) => typeof value !== "undefined")
              .reduce(
                (obj, [key, value]) => {
                  obj[key] = JSON.stringify(value);

                  return obj;
                },
                {} as Record<string, string>,
              ),
          },
        ],
        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      }),
    );

    await Promise.all([
      typeof req.roles !== "undefined"
        ? setRoles(c, auth, req.keyId, req.roles)
        : Promise.resolve(),
      typeof req.permissions !== "undefined"
        ? setPermissions(c, auth, req.keyId, req.permissions)
        : Promise.resolve(),
    ]);

    return c.json({});
  });
