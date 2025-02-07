import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { retry } from "@/pkg/util/retry";
import { DatabaseError } from "@planetscale/database";
import {
  type EncryptedKey,
  type Identity,
  type Key,
  type KeyPermission,
  type KeyRole,
  schema,
} from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["migrations"],
  operationId: "v1.migrations.createKeys",
  method: "post" as const,
  path: "/v1/migrations.createKeys",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z
            .array(
              z.object({
                apiId: z.string().openapi({
                  description: "Choose an `API` where this key should be created.",
                  example: "api_123",
                }),
                prefix: z
                  .string()
                  .max(8)
                  .optional()
                  .openapi({
                    description: `To make it easier for your users to understand which product an api key belongs to, you can add prefix them.

For example Stripe famously prefixes their customer ids with cus_ or their api keys with sk_live_.

The underscore is automatically added if you are defining a prefix, for example: "prefix": "abc" will result in a key like abc_xxxxxxxxx
`,
                  }),

                name: z.string().optional().openapi({
                  description: "The name for your Key. This is not customer facing.",
                  example: "my key",
                }),
                plaintext: z.string().optional().openapi({
                  description:
                    "The raw key in plaintext. If provided, unkey encrypts this value and stores it securely. Provide either `hash` or `plaintext`",
                }),
                hash: z
                  .object({
                    value: z.string().openapi({ description: "The hashed and encoded key" }),
                    variant: z.enum(["sha256_base64"]).openapi({
                      description:
                        "The algorithm for hashing and encoding, currently only sha256 and base64 are supported",
                    }),
                  })
                  .optional()
                  .openapi({
                    description: "Provide either `hash` or `plaintext`",
                  }),
                start: z.string().optional().openapi({
                  description:
                    "The first 4 characters of the key. If a prefix is used, it should be the prefix plus 4 characters.",
                  example: "unkey_32kq",
                }),
                ownerId: z.string().optional().openapi({
                  description: "Deprecated, use `externalId`",
                  example: "team_123",
                  deprecated: true,
                }),
                externalId: z
                  .string()
                  .optional()
                  .openapi({
                    description: `Your user’s Id. This will provide a link between Unkey and your customer record.
When validating a key, we will return this back to you, so you can clearly identify your user from their api key.`,
                    example: "user_123",
                  }),
                meta: z
                  .record(z.unknown())
                  .optional()
                  .openapi({
                    description:
                      "This is a place for dynamic meta data, anything that feels useful for you should go here",
                    example: {
                      billingTier: "PRO",
                      trialEnds: "2023-06-16T17:16:37.161Z",
                    },
                  }),
                roles: z
                  .array(z.string().min(1).max(512))
                  .optional()
                  .openapi({
                    description:
                      "A list of roles that this key should have. If the role does not exist, an error is thrown",
                    example: ["admin", "finance"],
                  }),
                permissions: z
                  .array(z.string().min(1).max(512))
                  .optional()
                  .openapi({
                    description:
                      "A list of permissions that this key should have. If the permission does not exist, an error is thrown",
                    example: ["domains.create_record", "say_hello"],
                  }),
                expires: z.number().int().optional().openapi({
                  description:
                    "You can auto expire keys by providing a unix timestamp in milliseconds. Once Keys expire they will automatically be disabled and are no longer valid unless you enable them again.",
                  example: 1623869797161,
                }),
                remaining: z
                  .number()
                  .int()
                  .min(1)
                  .optional()
                  .openapi({
                    description:
                      "You can limit the number of requests a key can make. Once a key reaches 0 remaining requests, it will automatically be disabled and is no longer valid unless you update it.",
                    example: 1000,
                    externalDocs: {
                      description: "Learn more",
                      url: "https://unkey.dev/docs/features/remaining",
                    },
                  }),
                refill: z
                  .object({
                    interval: z.enum(["daily", "monthly"]).openapi({
                      description:
                        "Unkey will automatically refill verifications at the set interval.",
                    }),
                    amount: z.number().int().min(1).positive().openapi({
                      description:
                        "The number of verifications to refill for each occurrence is determined individually for each key.",
                    }),
                    refillDay: z.number().min(1).max(31).optional().openapi({
                      description:
                        "The day verifications will refill each month, when interval is set to 'monthly'",
                    }),
                  })
                  .optional()
                  .openapi({
                    description:
                      "Unkey enables you to refill verifications for each key at regular intervals.",
                    example: {
                      interval: "daily",
                      amount: 100,
                    },
                  }),
                ratelimit: z
                  .object({
                    async: z
                      .boolean()
                      .default(false)
                      .optional()
                      .openapi({
                        description:
                          "Async will return a response immediately, lowering latency at the cost of accuracy.",
                        externalDocs: {
                          description: "Learn more",
                          url: "https://unkey.dev/docs/features/ratelimiting",
                        },
                      }),
                    type: z
                      .enum(["fast", "consistent"])
                      .default("fast")
                      .openapi({
                        deprecated: true,
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
                      deprecated: true,
                    }),
                    refillInterval: z.number().int().min(1).openapi({
                      description:
                        "Determines the speed at which tokens are refilled, in milliseconds.",
                      deprecated: true,
                    }),
                  })
                  .optional()
                  .openapi({
                    description: "Unkey comes with per-key ratelimiting out of the box.",
                    example: {
                      type: "fast",
                      limit: 10,
                      refillRate: 1,
                      refillInterval: 60,
                    },
                  }),
                enabled: z.boolean().default(true).optional().openapi({
                  description: "Sets if key is enabled or disabled. Disabled keys are not valid.",
                  example: false,
                }),
                environment: z
                  .string()
                  .max(256)
                  .optional()
                  .openapi({
                    description: `Environments allow you to divide your keyspace.

Some applications like Stripe, Clerk, WorkOS and others have a concept of "live" and "test" keys to
give the developer a way to develop their own application without the risk of modifying real world
resources.

When you set an environment, we will return it back to you when validating the key, so you can
handle it correctly.
              `,
                  }),
              }),
            )
            .max(100),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The key ids of all created keys",
      content: {
        "application/json": {
          schema: z.object({
            keyIds: z.array(z.string()).openapi({
              description:
                "The ids of the keys. This is not a secret and can be stored as a reference if you wish. You need the keyId to update or delete a key later.",
              example: ["key_123", "key_456"],
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1MigrationsCreateKeysRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1MigrationsCreateKeysResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1MigrationsCreateKeys = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");

    const { cache, db, rbac, vault, logger } = c.get("services");

    const auth = await rootKeyAuth(c);
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;

    const roleNames = req.filter((k) => k.roles).flatMap((k) => k.roles) as string[];
    // name -> id
    const roles: Record<string, string> = {};

    if (roleNames.length > 0) {
      const found = await db.primary.query.roles.findMany({
        where: (table, { inArray, and, eq }) =>
          and(eq(table.workspaceId, authorizedWorkspaceId), inArray(table.name, roleNames)),
      });
      const missingRoles = roleNames.filter((name) => !found.some((role) => role.name === name));
      if (missingRoles.length > 0) {
        throw new UnkeyApiError({
          code: "PRECONDITION_FAILED",
          message: `Roles ${JSON.stringify(missingRoles)} are missing, please create them first`,
        });
      }
      for (const role of found) {
        roles[role.name] = role.id;
      }
    }

    const permissionNames = req
      .filter((k) => k.permissions)
      .flatMap((k) => k.permissions) as string[];
    // name -> id
    const permissions: Record<string, string> = {};

    if (permissionNames.length > 0) {
      const found = await db.primary.query.permissions.findMany({
        where: (table, { inArray, and, eq }) =>
          and(eq(table.workspaceId, authorizedWorkspaceId), inArray(table.name, permissionNames)),
      });
      const missingPermissions = permissionNames.filter(
        (name) => !found.some((permission) => permission.name === name),
      );
      if (missingPermissions.length > 0) {
        throw new UnkeyApiError({
          code: "PRECONDITION_FAILED",
          message: `Permissions ${JSON.stringify(
            missingPermissions,
          )} are missing, please create them first`,
        });
      }
      for (const permission of found) {
        permissions[permission.name] = permission.id;
      }
    }

    const createIdentities: Array<Identity> = [];
    const keys: Array<Key> = [];
    const encryptedKeys: Array<EncryptedKey> = [];
    const roleConnections: Array<KeyRole> = [];
    const permissionConnections: Array<KeyPermission> = [];

    const requestWithKeyIds = req.map((r) => ({
      ...r,
      keyId: newId("key"),
    }));

    /**
     * Encrypt keys
     */

    await Promise.all(
      requestWithKeyIds.map(async (key) => {
        const perm = rbac.evaluatePermissions(
          buildUnkeyQuery(({ or }) =>
            or(
              "*",
              "api.*.create_key",
              `api.${key.apiId}.create_key`,
              key.plaintext ? "api.*.encrypt_key" : undefined,
              key.plaintext ? `api.${key.apiId}.encrypt_key` : undefined,
            ),
          ),
          auth.permissions,
        );
        if (perm.err) {
          throw new UnkeyApiError({
            code: "INTERNAL_SERVER_ERROR",
            message: perm.err.message,
          });
        }

        if (!perm.val.valid) {
          throw new UnkeyApiError({
            code: "INSUFFICIENT_PERMISSIONS",
            message: "unauthorized",
          });
        }

        const { val: api, err } = await cache.apiById.swr(key.apiId, async () => {
          return (
            (await db.readonly.query.apis.findFirst({
              where: (table, { eq, and, isNull }) =>
                and(
                  eq(table.workspaceId, authorizedWorkspaceId),
                  eq(table.id, key.apiId),
                  isNull(table.deletedAt),
                ),
              with: {
                keyAuth: true,
              },
            })) ?? null
          );
        });
        if (err) {
          throw new UnkeyApiError({
            code: "INTERNAL_SERVER_ERROR",
            message: `unable to load api: ${err.message}`,
          });
        }

        if (!api || api.workspaceId !== auth.authorizedWorkspaceId) {
          throw new UnkeyApiError({
            code: "NOT_FOUND",
            message: `api ${key.apiId} not found`,
          });
        }

        if (!api.keyAuthId) {
          throw new UnkeyApiError({
            code: "PRECONDITION_FAILED",
            message: `api ${key.apiId} is not setup to handle keys`,
          });
        }

        if ((key.remaining === null || key.remaining === undefined) && key.refill?.interval) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "remaining must be set if you are using refill.",
          });
        }

        if (!key.hash && !key.plaintext) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "provide either `hash` or `plaintext`",
          });
        }
        if (key.refill?.refillDay && key.refill.interval === "daily") {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "when interval is set to 'daily', 'refillDay' must be null.",
          });
        }

        let identityId: string | null = null;
        if (key.externalId) {
          const identity = await cache.identityByExternalId.swr(
            key.externalId,
            async (externalId) => {
              return await db.readonly.query.identities.findFirst({
                where: (table, { eq, and }) =>
                  and(
                    eq(table.workspaceId, authorizedWorkspaceId),
                    eq(table.externalId, externalId),
                  ),
              });
            },
          );
          if (identity.err) {
            throw new UnkeyApiError({
              code: "INTERNAL_SERVER_ERROR",
              message: `unable to load identity: ${identity.err.message}`,
            });
          }
          if (identity.val) {
            identityId = identity.val.id;
          } else {
            identityId = newId("identity");
            createIdentities.push({
              id: identityId,
              externalId: key.externalId,
              workspaceId: authorizedWorkspaceId,
              createdAt: Date.now(),
              updatedAt: null,
              meta: null,
              environment: "default",
            });
          }
        }

        /**
         * Set up an api for production
         */

        const hash = key.plaintext ? await sha256(key.plaintext) : key.hash!.value;

        keys.push({
          id: key.keyId,
          keyAuthId: api.keyAuthId!,
          name: key.name ?? null,
          hash: hash,
          start: key.start ?? key.plaintext?.slice(0, 4) ?? "",
          ownerId: key.ownerId ?? null,
          identityId,
          meta: key.meta ? JSON.stringify(key.meta) : null,
          workspaceId: authorizedWorkspaceId,
          forWorkspaceId: null,
          expires: key.expires ? new Date(key.expires) : null,
          createdAt: new Date(),
          ratelimitAsync: key.ratelimit?.async ?? key.ratelimit?.type === "fast",
          ratelimitLimit: key.ratelimit?.limit ?? key.ratelimit?.refillRate ?? null,
          ratelimitDuration: key.ratelimit?.refillInterval ?? key.ratelimit?.refillInterval ?? null,
          remaining: key.remaining ?? null,
          refillDay: key.refill?.interval === "daily" ? null : key?.refill?.refillDay ?? 1,
          refillAmount: key.refill?.amount ?? null,
          deletedAt: null,
          enabled: key.enabled ?? true,
          environment: key.environment ?? null,
          createdAtM: Date.now(),
          updatedAtM: null,
          deletedAtM: null,
          lastRefillAt: null,
        });

        for (const role of key.roles ?? []) {
          const roleId = roles[role];
          roleConnections.push({
            keyId: key.keyId,
            createdAt: new Date(),
            roleId,
            updatedAt: null,
            workspaceId: authorizedWorkspaceId,
          });
        }
        for (const permission of key.permissions ?? []) {
          const permissionId = permissions[permission];
          permissionConnections.push({
            keyId: key.keyId,
            createdAt: new Date(),
            permissionId,
            tempId: 0,
            updatedAt: null,
            workspaceId: authorizedWorkspaceId,
          });
        }

        if (key.plaintext) {
          const encryptionResponse = await retry(
            3,
            () =>
              vault.encrypt(c, {
                keyring: authorizedWorkspaceId,
                data: key.plaintext!,
              }),
            (attempt, err) =>
              logger.warn("vault.encrypt failed", {
                attempt,
                err: err.message,
              }),
          );
          encryptedKeys.push({
            workspaceId: authorizedWorkspaceId,
            keyId: key.keyId,
            encrypted: encryptionResponse.encrypted,
            encryptionKeyId: encryptionResponse.keyId,
          });
        }
      }),
    );

    await db.primary.transaction(async (tx) => {
      if (createIdentities.length > 0) {
        await tx
          .insert(schema.identities)
          .values(createIdentities)
          .onDuplicateKeyUpdate({ set: { updatedAt: Date.now() } })
          .catch((e) => {
            logger.error("unable to create identities", {
              error: e.message,
            });
            throw new UnkeyApiError({
              code: "INTERNAL_SERVER_ERROR",
              message: "unable to create identities",
            });
          });
      }

      if (keys.length > 0) {
        await tx
          .insert(schema.keys)
          .values(keys)
          .catch((e) => {
            if (e instanceof DatabaseError && e.body.message.includes("Duplicate entry")) {
              logger.warn("migrating duplicate key", {
                error: e.body.message,
                workspaceId: authorizedWorkspaceId,
              });
              throw new UnkeyApiError({
                code: "NOT_UNIQUE",
                message: e.body.message,
              });
            }
            throw e;
          });
      }
      if (encryptedKeys.length > 0) {
        await tx.insert(schema.encryptedKeys).values(encryptedKeys);
      }
      if (roleConnections.length > 0) {
        await tx.insert(schema.keysRoles).values(roleConnections);
      }
      if (permissionConnections.length > 0) {
        await tx.insert(schema.keysPermissions).values(permissionConnections);
      }

      await insertUnkeyAuditLog(
        c,
        tx,
        keys.map((k) => ({
          workspaceId: authorizedWorkspaceId,
          event: "key.create",
          actor: {
            type: "key",
            id: auth.key.id,
          },
          description: `Created ${k.id} in ${k.keyAuthId}`,
          resources: [
            {
              type: "key",
              id: k.id,
            },
            {
              type: "keyAuth",
              id: k.keyAuthId!,
            },
          ],

          context: {
            location: c.get("location"),
            userAgent: c.get("userAgent"),
          },
        })),
      );
    });

    return c.json({
      keyIds: requestWithKeyIds.map((k) => k.keyId),
    });
  });
