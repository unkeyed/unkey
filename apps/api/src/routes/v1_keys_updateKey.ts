import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { type Credits, type Key, and, schema } from "@unkey/db";
import { eq } from "@unkey/db";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";
import { upsertIdentity } from "./v1_keys_createKey";
import { setPermissions } from "./v1_keys_setPermissions";
import { setRoles } from "./v1_keys_setRoles";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "updateKey",
  summary: "Update key settings",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
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
              ownerId: z
                .string()
                .nullish()
                .openapi({
                  deprecated: true,
                  description: `Deprecated, use \`externalId\`
                    The id of the tenant associated with this key. Use whatever reference you have in your system to identify the tenant. When verifying the key, we will send this field back to you, so you know who is accessing your API.`,
                  example: "user_123",
                }),
              externalId: z
                .string()
                .nullish()
                .openapi({
                  description: `The id of the tenant associated with this key. Use whatever reference you have in your system to identify the tenant. When verifying the key, we will send this back to you, so you know who is accessing your API.
                  Under the hood this upserts and connects an \`ìdentity\` for you.
                  To disconnect the key from an identity, set \`externalId: null\`.`,
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
                  refillDay: z.number().min(1).max(31).optional().openapi({
                    description:
                      "The day verifications will refill each month, when interval is set to 'monthly'",
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
    const { cache, db, usageLimiter, rbac } = c.get("services");
    const auth = await rootKeyAuth(c);
    const key = await db.primary.query.keys.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.id, req.keyId), isNull(table.deletedAtM)),
      with: {
        credits: true,
        keyAuth: {
          with: {
            api: true,
          },
        },
        ratelimits: true,
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
    if (req.refill?.interval === "daily" && req.refill.refillDay) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "Cannot set 'refillDay' if 'interval' is 'daily'",
      });
    }
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;

    const changes: Partial<Key> = {};
    const creditChanges: Partial<Credits> = {};

    const hasOldCredits = key.remaining !== null;
    const hasNewCredits = key.credits !== null;

    if (typeof req.name !== "undefined") {
      changes.name = req.name;
    }

    if (typeof req.meta !== "undefined") {
      changes.meta = req.meta === null ? null : JSON.stringify(req.meta);
    }

    if (typeof req.externalId !== "undefined") {
      if (req.externalId === null) {
        changes.identityId = null;
        changes.ownerId = null;
      } else {
        const identity = await upsertIdentity(db.primary, authorizedWorkspaceId, req.externalId);
        changes.identityId = identity.id;
        changes.ownerId = req.externalId;
      }
    } else if (typeof req.ownerId !== "undefined") {
      if (req.ownerId === null) {
        changes.identityId = null;
        changes.ownerId = null;
      } else {
        const identity = await upsertIdentity(db.primary, authorizedWorkspaceId, req.ownerId);
        changes.identityId = identity.id;
        changes.ownerId = req.ownerId;
      }
    }

    if (typeof req.expires !== "undefined") {
      changes.expires = req.expires === null ? null : new Date(req.expires);
    }

    let creditDeleted = false;
    let creditCreatedId: string | undefined = undefined;

    if (typeof req.remaining !== "undefined") {
      // Key has new credit system.
      if (hasNewCredits) {
        if (req.remaining === null) {
          await db.primary.delete(schema.credits).where(eq(schema.credits.id, key.credits.id));
          creditDeleted = true;
        } else {
          creditChanges.remaining = req.remaining as number;
        }
      } else if (hasOldCredits) {
        // Key has old credit system, so we work based off that
        changes.remaining = req.remaining;
      } else {
        // Key doesn't have old system so we use new system from the getgo
        const newCreditId = newId("credit");
        creditCreatedId = newCreditId;
        // Use upsert to prevent race condition with concurrent updates
        await db.primary
          .insert(schema.credits)
          .values({
            id: newCreditId,
            keyId: key.id,
            workspaceId: auth.authorizedWorkspaceId,
            createdAt: Date.now(),
            refilledAt: Date.now(),
            remaining: req.remaining as number,
            identityId: null,
            refillAmount: req.refill?.amount ?? null,
            refillDay: req.refill
              ? req.refill.interval === "monthly"
                ? (req.refill.refillDay ?? 1)
                : null
              : null,
            updatedAt: null,
          })
          .onDuplicateKeyUpdate({
            set: {
              remaining: req.remaining as number,
              refillAmount: req.refill?.amount ?? null,
              refillDay: req.refill
                ? req.refill.interval === "monthly"
                  ? (req.refill.refillDay ?? 1)
                  : null
                : null,
              refilledAt: req.refill?.interval ? Date.now() : null,
              updatedAt: Date.now(),
            },
          });
      }
    }

    if (typeof req.ratelimit !== "undefined") {
      if (req.ratelimit === null) {
        await db.primary
          .delete(schema.ratelimits)
          .where(
            and(
              eq(schema.ratelimits.workspaceId, auth.authorizedWorkspaceId),
              eq(schema.ratelimits.name, "default"),
              eq(schema.ratelimits.keyId, key.id),
            ),
          );
      } else {
        const existing = key.ratelimits.find((r) => r.name === "default");
        if (existing) {
          await db.primary
            .update(schema.ratelimits)
            .set({
              limit: req.ratelimit.limit ?? req.ratelimit.refillRate,
              duration: req.ratelimit.duration ?? req.ratelimit.refillInterval,
            })
            .where(and(eq(schema.ratelimits.id, existing.id)));
        } else {
          await db.primary.insert(schema.ratelimits).values({
            id: newId("ratelimit"),
            workspaceId: auth.authorizedWorkspaceId,
            name: "default",
            keyId: key.id,
            limit: req.ratelimit.limit ?? req.ratelimit.refillRate!,
            duration: req.ratelimit.duration ?? req.ratelimit.refillInterval!,
            autoApply: true,
          });
        }
      }
    }

    if (typeof req.refill !== "undefined") {
      if (hasNewCredits || !hasOldCredits) {
        if (req.refill === null) {
          creditChanges.refillAmount = null;
          creditChanges.refillDay = null;
          creditChanges.refilledAt = null;
        } else {
          creditChanges.refillAmount = req.refill.amount;
          creditChanges.refillDay =
            req.refill.interval === "monthly" ? (req.refill.refillDay ?? 1) : null;
        }
      }

      if (hasOldCredits) {
        if (req.refill === null) {
          changes.refillAmount = null;
          changes.refillDay = null;
          changes.lastRefillAt = null;
        } else {
          changes.refillAmount = req.refill.amount;
          changes.refillDay =
            req.refill.interval === "monthly" ? (req.refill.refillDay ?? 1) : null;
        }
      }
    }

    if (typeof req.enabled !== "undefined") {
      changes.enabled = req.enabled;
    }

    if (Object.keys(changes).length) {
      await db.primary.update(schema.keys).set(changes).where(eq(schema.keys.id, key.id));
    }

    // Since we don't know if we already have a row for this key we just update on keyId
    // Skip if we deleted the credits row earlier
    if (Object.keys(creditChanges).length && !creditDeleted) {
      await db.primary
        .update(schema.credits)
        .set(creditChanges)
        .where(eq(schema.credits.keyId, key.id));
    }

    await insertUnkeyAuditLog(c, undefined, {
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
    });

    // Revalidate usage limiter with appropriate ID based on which credit system is in use
    // Determine credit state from in-memory tracking to avoid stale reads from db.readonly
    let creditIdForRevalidation: string | undefined;
    if (creditCreatedId) {
      // A new credit was created
      creditIdForRevalidation = creditCreatedId;
    } else if (hasNewCredits && !creditDeleted) {
      // Credit existed and wasn't deleted
      creditIdForRevalidation = key.credits.id;
    }
    // else: credit was deleted or never existed, use keyId

    c.executionCtx.waitUntil(
      usageLimiter.revalidate(
        creditIdForRevalidation ? { creditId: creditIdForRevalidation } : { keyId: key.id },
      ),
    );
    c.executionCtx.waitUntil(cache.keyByHash.remove(key.hash));
    c.executionCtx.waitUntil(cache.keyById.remove(key.id));

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
