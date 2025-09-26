import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { retry } from "@/pkg/util/retry";
import type { QueueContentType } from "@cloudflare/workers-types";
import { sha256 } from "@unkey/hash";
import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  deprecated: true,
  tags: ["migrations"],
  operationId: "v1.migrations.enqueueKeys",
  summary: "Enqueue key migration",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "post" as const,
  path: "/v1/migrations.enqueueKeys",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            migrationId: z.string().openapi({
              description: "Contact support@unkey.dev to receive your migration id.",
            }),
            apiId: z.string().openapi({
              description: "The id of the api, you want to migrate keys to",
            }),
            keys: z
              .array(
                z.object({
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
                        .default(true)
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
                      duration: z.number().int().min(1000).openapi({
                        description: "The window duration in milliseconds",
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
                      description:
                        "Unkey comes with per-key fixed-window ratelimiting out of the box.",
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
          }),
        },
      },
    },
  },
  responses: {
    202: {
      description: "The key ids of all created keys",
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
export type V1MigrationsEnqueueKeysRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1MigrationsEnqueueKeysResponse = z.infer<
  (typeof route.responses)[202]["content"]["application/json"]["schema"]
>;

export const registerV1MigrationsEnqueueKeys = (app: App) =>
  app.openapi(route, async (c) => {
    if (!c.env.KEY_MIGRATIONS) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: "queue migrations are not enabled",
      });
    }
    const req = c.req.valid("json");
    const { db, logger, vault, cache } = c.get("services");

    const withEncryption = req.keys.some((r) => typeof r.plaintext === "string");

    const auth = await rootKeyAuth(
      c,
      buildUnkeyQuery(({ or, and }) =>
        or(
          "*",
          and(
            or("api.*.create_key", `api.${req.apiId}.create_key`),
            withEncryption ? or("api.*.encrypt_key", `api.${req.apiId}.encrypt_key`) : undefined,
          ),
        ),
      ),
    );
    const authorizedWorkspaceId = auth.authorizedWorkspaceId;

    const api = await cache.apiById.swr(req.apiId, async () =>
      db.readonly.query.apis.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, authorizedWorkspaceId), eq(table.id, req.apiId)),
        with: {
          keyAuth: true,
        },
      }),
    );
    if (api.err) {
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: api.err.message,
      });
    }
    if (!api.val) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "api not found",
      });
    }
    if (!api.val.keyAuth) {
      throw new UnkeyApiError({
        code: "NOT_FOUND",
        message: "keyauth not found",
      });
    }

    const encrypted: Record<string, { encrypted: string; keyId: string }> = {};
    const toBeEncrypted = req.keys.filter((k) => k.plaintext).map((k) => k.plaintext!);

    if (toBeEncrypted.length > 0) {
      const bulkEncryptionRes = await retry(5, () =>
        vault.encryptBulk(c, {
          keyring: authorizedWorkspaceId,
          data: toBeEncrypted,
        }),
      );
      for (let i = 0; i < bulkEncryptionRes.encrypted.length; i++) {
        encrypted[toBeEncrypted[i]] = {
          keyId: bulkEncryptionRes.encrypted[i].keyId,
          encrypted: bulkEncryptionRes.encrypted[i].encrypted,
        };
      }
    }

    await c.env.KEY_MIGRATIONS.sendBatch(
      await Promise.all(
        req.keys.map(async (k) => {
          let hash = k.hash?.value;
          if (!hash) {
            if (!k.plaintext) {
              throw new UnkeyApiError({
                code: "BAD_REQUEST",
                message: "Either plaintext or hash must be provided",
              });
            }
            hash = await sha256(k.plaintext);
          }

          return {
            contentType: "json" as QueueContentType,
            body: {
              migrationId: req.migrationId,
              keyAuthId: api.val!.keyAuth!.id,
              workspaceId: auth.authorizedWorkspaceId,
              rootKeyId: auth.key.id,
              auditLogContext: {
                location: c.get("location"),
                userAgent: c.get("userAgent") ?? "",
              },
              prefix: k.prefix,
              name: k.name,
              hash: hash!,
              start: k.start,
              ownerId: k.ownerId,
              meta: k.meta,
              roles: k.roles,
              permissions: k.permissions,
              expires: k.expires,
              remaining: k.remaining,
              refill: k.refill,
              ratelimit: k.ratelimit
                ? {
                    async: k.ratelimit.async ?? k.ratelimit.type === "fast",
                    limit: k.ratelimit.limit ?? k.ratelimit.refillRate,
                    duration: k.ratelimit.duration ?? k.ratelimit.refillInterval,
                  }
                : undefined,
              enabled: k.enabled ?? true,
              environment: k.environment,
              encrypted: k.plaintext ? encrypted[k.plaintext] : undefined,
            },
          };
        }),
      ),
    ).catch((e) => {
      logger.error("Failed to enqueue keys", { error: (e as Error).message });
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to enqueue keys",
      });
    });

    return c.json({}, { status: 202 });
  });
