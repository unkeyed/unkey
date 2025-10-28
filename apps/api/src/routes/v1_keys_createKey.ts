import { type UnkeyAuditLog, insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import type { Database, Identity } from "@/pkg/db";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { retry } from "@/pkg/util/retry";
import { revalidateKeyCount } from "@/pkg/util/revalidate_key_count";
import { createRoute, z } from "@hono/zod-openapi";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { type RBAC, buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["keys"],
  operationId: "createKey",
  summary: "Create API key",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post" as const,
  path: "/v1/keys.createKey",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            apiId: z.string().openapi({
              description: "Choose an `API` where this key should be created.",
              example: "api_123",
            }),
            prefix: z
              .string()
              .max(16)
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
            byteLength: z.number().int().min(16).max(255).default(16).optional().openapi({
              description:
                "The byte length used to generate your key determines its entropy as well as its length. Higher is better, but keys become longer and more annoying to handle. The default is 16 bytes, or 2^^128 possible combinations.",
              default: 16,
            }),
            ownerId: z.string().optional().openapi({
              deprecated: true,
              description: "Deprecated, use `externalId`",
              example: "team_123",
            }),
            externalId: z
              .string()
              .optional()
              .openapi({
                description: `Your user's Id. This will provide a link between Unkey and your customer record.
When validating a key, we will return this back to you, so you can clearly identify your user from their api key.`,
                example: "team_123",
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
                  description: "Unkey will automatically refill verifications at the set interval.",
                }),
                amount: z.number().int().min(1).positive().openapi({
                  description:
                    "The number of verifications to refill for each occurrence is determined individually for each key.",
                }),
                refillDay: z
                  .number()
                  .min(1)
                  .max(31)
                  .optional()
                  .openapi({
                    description: `The day of the month, when we will refill the remaining verifications. To refill on the 15th of each month, set 'refillDay': 15.
                    If the day does not exist, for example you specified the 30th and it's february, we will refill them on the last day of the month instead.`,
                  }),
              })
              .optional()
              .openapi({
                description:
                  "Unkey enables you to refill verifications for each key at regular intervals.",
                example: {
                  interval: "monthly",
                  amount: 100,
                  refillDay: 15,
                },
              }),

            ratelimit: z
              .object({
                async: z
                  .boolean()
                  .default(true)
                  .optional()
                  .openapi({
                    description:
                      "Async will return a response immediately, lowering latency at the cost of accuracy. Will be required soon.",
                    externalDocs: {
                      description: "Learn more",
                      url: "https://unkey.dev/docs/features/ratelimiting",
                    },
                  }),
                type: z
                  .enum(["fast", "consistent"])
                  .default("fast")
                  .optional()
                  .openapi({
                    description:
                      "Deprecated, use `async`. Fast ratelimiting doesn't add latency, while consistent ratelimiting is more accurate.",
                    externalDocs: {
                      description: "Learn more",
                      url: "https://unkey.dev/docs/features/ratelimiting",
                    },
                    deprecated: true,
                  }),
                limit: z.number().int().min(1).openapi({
                  description: "The total amount of requests in a given interval.",
                }),
                duration: z.number().int().min(1000).optional().openapi({
                  description: "The window duration in milliseconds. Will be required soon.",
                  example: 60_000,
                }),

                refillRate: z.number().int().min(1).optional().openapi({
                  description: "How many tokens to refill during each refillInterval.",
                  deprecated: true,
                }),
                refillInterval: z.number().int().min(1).optional().openapi({
                  description: "The refill timeframe, in milliseconds.",
                  deprecated: true,
                }),
              })
              .optional()
              .openapi({
                description: "Unkey comes with per-key fixed-window ratelimiting out of the box.",
                example: {
                  type: "fast",
                  limit: 10,
                  duration: 60_000,
                },
              }),
            enabled: z.boolean().default(true).optional().openapi({
              description: "Sets if key is enabled or disabled. Disabled keys are not valid.",
              example: false,
            }),
            recoverable: z
              .boolean()
              .default(false)
              .optional()
              .openapi({
                description: `You may want to show keys again later. While we do not recommend this, we leave this option open for you.

In addition to storing the key's hash, recoverable keys are stored in an encrypted vault, allowing you to retrieve and display the plaintext later.

[https://www.unkey.com/docs/security/recovering-keys](https://www.unkey.com/docs/security/recovering-keys) for more information.`,
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
        },
      },
    },
  },
  responses: {
    200: {
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().openapi({
              description:
                "The id of the key. This is not a secret and can be stored as a reference if you wish. You need the keyId to update or delete a key later.",
              example: "key_123",
            }),
            key: z.string().openapi({
              description:
                "The newly created api key, do not store this on your own system but pass it along to your user.",
              example: "prefix_xxxxxxxxx",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysCreateKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysCreateKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysCreateKey = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const { cache, db, logger, vault, rbac } = c.get("services");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or }) => or("*", "api.*.create_key", `api.${req.apiId}.create_key`)),
    );

    const { val: api, err } = await cache.apiById.swr(req.apiId, async () => {
      return (
        (await db.readonly.query.apis.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.id, req.apiId), isNull(table.deletedAtM)),
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
        message: `api ${req.apiId} not found`,
      });
    }

    if (!api.keyAuth) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${req.apiId} is not setup to handle keys`,
      });
    }

    if (req.recoverable && !api.keyAuth.storeEncryptedKeys) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${req.apiId} does not support recoverable keys`,
      });
    }

    if (req.remaining === 0) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "remaining must be greater than 0.",
      });
    }

    if ((req.remaining === null || req.remaining === undefined) && req.refill?.interval) {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "remaining must be set if you are using refill.",
      });
    }

    if (req.refill?.refillDay && req.refill.interval === "daily") {
      throw new UnkeyApiError({
        code: "BAD_REQUEST",
        message: "when interval is set to 'daily', 'refillDay' must be null.",
      });
    }
    /**
     * Set up an api for production
     */

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    const externalId = req.externalId ?? req.ownerId;

    const [permissionIds, roleIds, identity] = await Promise.all([
      getPermissionIds(auth, rbac, db.primary, authorizedWorkspaceId, req.permissions ?? []),
      getRoleIds(auth, rbac, db.primary, authorizedWorkspaceId, req.roles ?? []),
      externalId
        ? upsertIdentity(db.primary, authorizedWorkspaceId, externalId)
        : Promise.resolve(null),
    ]);

    const newKey = await retry(5, async (attempt) => {
      if (attempt > 1) {
        logger.warn("retrying key creation", {
          attempt,
          workspaceId: authorizedWorkspaceId,
          apiId: api.id,
        });
      }

      const secret = new KeyV1({
        byteLength: req.byteLength ?? api.keyAuth?.defaultBytes ?? 16,
        prefix: req.prefix ?? (api.keyAuth?.defaultPrefix as string | undefined),
      }).toString();

      const start = secret.slice(0, (req.prefix?.length ?? 0) + 5);
      const kId = newId("key");
      const hash = await sha256(secret.toString());
      await db.primary.insert(schema.keys).values({
        id: kId,
        keyAuthId: api.keyAuthId!,
        name: req.name,
        hash,
        start,
        ownerId: externalId,
        meta: req.meta ? JSON.stringify(req.meta) : null,
        workspaceId: authorizedWorkspaceId,
        forWorkspaceId: null,
        expires: req.expires ? new Date(req.expires) : null,
        createdAtM: Date.now(),
        updatedAtM: null,
        enabled: req.enabled,
        environment: req.environment ?? null,
        identityId: identity?.id,
      });

      if (req.remaining) {
        await db.primary.insert(schema.credits).values({
          id: newId("credit"),
          remaining: req.remaining,
          workspaceId: authorizedWorkspaceId,
          createdAt: Date.now(),
          keyId: kId,
          identityId: null,
          refillAmount: req.refill?.amount,
          refillDay: req.refill
            ? req.refill.interval === "daily"
              ? null
              : (req.refill.refillDay ?? 1)
            : null,
          refilledAt: req.refill?.interval ? Date.now() : null,
          updatedAt: null,
        });
      }

      if (req.ratelimit) {
        await db.primary.insert(schema.ratelimits).values({
          id: newId("ratelimit"),
          keyId: kId,
          limit: req.ratelimit.limit ?? req.ratelimit.refillRate!,
          duration: req.ratelimit.duration ?? req.ratelimit.refillInterval!,
          workspaceId: authorizedWorkspaceId,
          name: "default",
          autoApply: true,
          identityId: null,
          createdAt: Date.now(),
        });
      }

      if (req.recoverable && api.keyAuth?.storeEncryptedKeys) {
        const perm = rbac.evaluatePermissions(
          buildUnkeyQuery(({ or }) => or("*", "api.*.encrypt_key", `api.${api.id}.encrypt_key`)),
          auth.permissions,
        );
        if (perm.err) {
          throw new UnkeyApiError({
            code: "INTERNAL_SERVER_ERROR",
            message: `unable to evaluate permissions: ${perm.err.message}`,
          });
        }
        if (!perm.val.valid) {
          throw new UnkeyApiError({
            code: "INSUFFICIENT_PERMISSIONS",
            message: `insufficient permissions to encrypt keys: ${perm.val.message}`,
          });
        }

        const vaultRes = await retry(
          3,
          async () => {
            return await vault.encrypt(c, {
              keyring: authorizedWorkspaceId,
              data: secret,
            });
          },
          (attempt, err) =>
            logger.warn("vault.encrypt failed", {
              attempt,
              err: err.message,
            }),
        );

        await db.primary.insert(schema.encryptedKeys).values({
          workspaceId: authorizedWorkspaceId,
          keyId: kId,
          encrypted: vaultRes.encrypted,
          encryptionKeyId: vaultRes.keyId,
        });
      }

      c.executionCtx.waitUntil(revalidateKeyCount(db.primary, api.keyAuthId!));

      return {
        id: kId,
        secret,
      };
    });

    await Promise.all([
      roleIds.length > 0
        ? db.primary.insert(schema.keysRoles).values(
            roleIds.map((roleId) => ({
              keyId: newKey.id,
              roleId,
              workspaceId: authorizedWorkspaceId,
            })),
          )
        : Promise.resolve(),
      permissionIds.length > 0
        ? db.primary.insert(schema.keysPermissions).values(
            permissionIds.map((permissionId) => ({
              keyId: newKey.id,
              permissionId,
              workspaceId: authorizedWorkspaceId,
            })),
          )
        : Promise.resolve(),
    ]);

    const auditLogs: UnkeyAuditLog[] = [
      {
        workspaceId: authorizedWorkspaceId,
        event: "key.create",
        actor: {
          type: "key",
          id: rootKeyId,
        },
        description: `Created ${newKey.id} in ${api.keyAuthId}`,
        resources: [
          {
            type: "key",
            id: newKey.id,
          },
          {
            type: "keyAuth",
            id: api.keyAuthId!,
          },
        ],

        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      },
      ...roleIds.map(
        (roleId) =>
          ({
            workspaceId: authorizedWorkspaceId,
            actor: { type: "key", id: rootKeyId },
            event: "authorization.connect_role_and_key",
            description: `Connected ${roleId} and ${newKey.id}`,
            resources: [
              {
                type: "key",
                id: newKey.id,
              },
              {
                type: "role",
                id: roleId,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          }) satisfies UnkeyAuditLog,
      ),
      ...permissionIds.map(
        (permissionId) =>
          ({
            workspaceId: authorizedWorkspaceId,
            actor: { type: "key", id: rootKeyId },
            event: "authorization.connect_permission_and_key",
            description: `Connected ${permissionId} and ${newKey.id}`,
            resources: [
              {
                type: "key",
                id: newKey.id,
              },
              {
                type: "permission",
                id: permissionId,
              },
            ],
            context: {
              location: c.get("location"),
              userAgent: c.get("userAgent"),
            },
          }) satisfies UnkeyAuditLog,
      ),
    ];

    await insertUnkeyAuditLog(c, undefined, auditLogs);
    return c.json({
      keyId: newKey.id,
      key: newKey.secret,
    });
  });

async function getPermissionIds(
  auth: { permissions: Array<string> },
  rbac: RBAC,
  db: Database,
  workspaceId: string,
  permissionsSlugs: Array<string>,
): Promise<Array<string>> {
  if (permissionsSlugs.length === 0) {
    return [];
  }
  const permissions = await db.query.permissions.findMany({
    where: (table, { inArray, and, eq }) =>
      and(eq(table.workspaceId, workspaceId), inArray(table.slug, permissionsSlugs)),
    columns: {
      id: true,
      slug: true,
    },
  });
  if (permissions.length === permissionsSlugs.length) {
    return permissions.map((r) => r.id);
  }

  const { val, err } = rbac.evaluatePermissions(
    buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_permission")),
    auth.permissions,
  );
  if (err) {
    throw new UnkeyApiError({
      code: "INTERNAL_SERVER_ERROR",
      message: `Failed to evaluate permissions: ${err.message}`,
    });
  }
  if (!val.valid) {
    throw new UnkeyApiError({
      code: "INSUFFICIENT_PERMISSIONS",
      message: val.message,
    });
  }

  const missingPermissionSlugs = permissionsSlugs.filter(
    (slug) => !permissions.some((permission) => permission.slug === slug),
  );

  const newPermissions = missingPermissionSlugs.map((slug) => ({
    id: newId("permission"),
    workspaceId,
    name: slug,
    slug,
  }));

  await db.insert(schema.permissions).values(newPermissions);

  return [...permissions, ...newPermissions].map((permission) => permission.id);
}

async function getRoleIds(
  auth: { permissions: Array<string> },
  rbac: RBAC,
  db: Database,
  workspaceId: string,
  roleNames: Array<string>,
): Promise<Array<string>> {
  if (roleNames.length === 0) {
    return [];
  }
  const roles = await db.query.roles.findMany({
    where: (table, { inArray, and, eq }) =>
      and(eq(table.workspaceId, workspaceId), inArray(table.name, roleNames)),
    columns: {
      id: true,
      name: true,
    },
  });
  if (roles.length === roleNames.length) {
    return roles.map((r) => r.id);
  }

  const { val, err } = rbac.evaluatePermissions(
    buildUnkeyQuery(({ or }) => or("*", "rbac.*.create_role")),
    auth.permissions,
  );
  if (err) {
    throw new UnkeyApiError({
      code: "INTERNAL_SERVER_ERROR",
      message: `Failed to evaluate permissions: ${err.message}`,
    });
  }
  if (!val.valid) {
    throw new UnkeyApiError({
      code: "INSUFFICIENT_PERMISSIONS",
      message: val.message,
    });
  }

  const missingRoles = roleNames.filter((name) => !roles.some((role) => role.name === name));

  const newRoles = missingRoles.map((name) => ({
    id: newId("role"),
    workspaceId,
    name,
  }));

  await db.insert(schema.roles).values(newRoles);

  return [...roles, ...newRoles].map((role) => role.id);
}

export async function upsertIdentity(
  db: Database,
  workspaceId: string,
  externalId: string,
): Promise<Identity> {
  let identity = await db.query.identities.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, workspaceId), eq(table.externalId, externalId)),
  });
  if (identity) {
    return identity;
  }

  await db
    .insert(schema.identities)
    .values({
      id: newId("identity"),
      createdAt: Date.now(),
      updatedAt: null,

      environment: "default",
      meta: {},
      externalId,
      workspaceId,
    })
    .onDuplicateKeyUpdate({
      set: {
        updatedAt: Date.now(),
      },
    });

  identity = await db.query.identities.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, workspaceId), eq(table.externalId, externalId)),
  });
  if (!identity) {
    throw new UnkeyApiError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to read identity after upsert",
    });
  }
  return identity;
}
