import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["migrations"],
  operationId: "v1.migrations.createKeys",
  method: "post",
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
                ownerId: z
                  .string()
                  .optional()
                  .openapi({
                    description: `Your user’s Id. This will provide a link between Unkey and your customer record.
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
                    type: z
                      .enum(["fast", "consistent"])
                      .default("fast")
                      .openapi({
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
    const { cache, db, analytics, rbac, vault } = c.get("services");

    const auth = await rootKeyAuth(c);

    const keyIds: Array<string> = [];
    await db.primary.transaction(async (tx) => {
      for (const key of req) {
        const perm = rbac.evaluatePermissions(
          buildUnkeyQuery(({ or }) => or("*", "api.*.create_key", `api.${key.apiId}.create_key`)),
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
                and(eq(table.id, key.apiId), isNull(table.deletedAt)),
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

        if (!!key.hash && key.plaintext) {
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: "provide either `hash` or `plaintext`",
          });
        }
        /**
         * Set up an api for production
         */
        const authorizedWorkspaceId = auth.authorizedWorkspaceId;

        let encrypted: string | null = null;
        if (key.plaintext) {
          const encryptionResponse = await vault.encrypt({
            keyring: authorizedWorkspaceId,
            data: key.plaintext,
          });
          encrypted = encryptionResponse.encrypted;
        }

        const keyId = newId("key");

        const rootKeyId = auth.key.id;

        let roleIds: string[] = [];
        if (key.roles && key.roles.length > 0) {
          const roles = await tx.query.roles.findMany({
            where: (table, { inArray, and, eq }) =>
              and(eq(table.workspaceId, authorizedWorkspaceId), inArray(table.name, key.roles!)),
          });
          if (roles.length < key.roles.length) {
            const missingRoles = key.roles.filter(
              (name) => !roles.some((role) => role.name === name),
            );
            throw new UnkeyApiError({
              code: "PRECONDITION_FAILED",
              message: `Roles ${JSON.stringify(
                missingRoles,
              )} are missing, please create them first`,
            });
          }
          roleIds = roles.map((r) => r.id);
        }

        const hash = key.plaintext ? await sha256(key.plaintext) : key.hash!.value;

        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: api.keyAuthId!,
          name: key.name,
          hash: hash,
          start: key.start ?? key.plaintext?.slice(0, 4) ?? "",
          ownerId: key.ownerId,
          meta: key.meta ? JSON.stringify(key.meta) : null,
          workspaceId: authorizedWorkspaceId,
          forWorkspaceId: null,
          expires: key.expires ? new Date(key.expires) : null,
          createdAt: new Date(),
          ratelimitLimit: key.ratelimit?.limit,
          ratelimitRefillRate: key.ratelimit?.refillRate,
          ratelimitRefillInterval: key.ratelimit?.refillInterval,
          ratelimitType: key.ratelimit?.type,
          remaining: key.remaining,
          refillInterval: key.refill?.interval,
          refillAmount: key.refill?.amount,
          lastRefillAt: key.refill?.interval ? new Date() : null,
          deletedAt: null,
          enabled: key.enabled,
          environment: key.environment ?? null,
          encrypted,
        });
        keyIds.push(keyId);

        await analytics.ingestUnkeyAuditLogs({
          workspaceId: authorizedWorkspaceId,
          event: "key.create",
          actor: {
            type: "key",
            id: rootKeyId,
          },
          description: `Created ${keyId} in ${api.keyAuthId}`,
          resources: [
            {
              type: "key",
              id: keyId,
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
        });
        if (roleIds.length > 0) {
          await tx.insert(schema.keysRoles).values(
            roleIds.map((roleId) => ({
              keyId,
              roleId,
              workspaceId: authorizedWorkspaceId,
            })),
          );
          await analytics.ingestUnkeyAuditLogs(
            roleIds.map((roleId) => ({
              workspaceId: authorizedWorkspaceId,
              actor: { type: "key", id: rootKeyId },
              event: "authorization.connect_role_and_key",
              description: `Connected ${roleId} and ${keyId}`,
              resources: [
                {
                  type: "key",
                  id: keyId,
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
            })),
          );
        }
      }
    });
    return c.json({
      keyIds,
    });
  });
