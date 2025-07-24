import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { insertUnkeyAuditLog } from "@/pkg/audit";
import { rootKeyAuth } from "@/pkg/auth/root_key";
import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

const route = createRoute({
  operationId: "deprecated.createKey",
  summary: "Create key (deprecated)",
  "x-speakeasy-ignore": true,
  "x-excluded": true,

  method: "post",
  path: "/v1/keys",
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
            byteLength: z.number().int().min(16).max(255).optional().default(16).openapi({
              description:
                "The byte length used to generate your key determines its entropy as well as its length. Higher is better, but keys become longer and more annoying to handle. The default is 16 bytes, or 2^^128 possible combinations.",
              default: 16,
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
export type LegacyKeysCreateKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type LegacyKeysCreateKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyKeysCreate = (app: App) =>
  app.openapi(route, async (c) => {
    const { cache, db } = c.get("services");
    const auth = await rootKeyAuth(c);

    const req = c.req.valid("json");

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

    if (!api.keyAuthId) {
      throw new UnkeyApiError({
        code: "PRECONDITION_FAILED",
        message: `api ${req.apiId} is not setup to handle keys`,
      });
    }

    /**
     * Set up an api for production
     */
    const key = new KeyV1({
      byteLength: req.byteLength,
      prefix: req.prefix,
    }).toString();
    const start = key.slice(0, (req.prefix?.length ?? 0) + 5);
    const keyId = newId("key");
    const hash = await sha256(key.toString());

    const authorizedWorkspaceId = auth.authorizedWorkspaceId;
    const rootKeyId = auth.key.id;
    await db.primary.transaction(async (tx) => {
      await tx.insert(schema.keys).values({
        id: keyId,
        // biome-ignore lint/style/noNonNullAssertion: This is legacy already no need for change
        keyAuthId: api.keyAuthId!,
        name: req.name,
        hash,
        start,
        ownerId: req.ownerId,
        meta: req.meta ? JSON.stringify(req.meta) : null,
        workspaceId: authorizedWorkspaceId,
        forWorkspaceId: null,
        expires: req.expires ? new Date(req.expires) : null,
        createdAtM: Date.now(),
        remaining: req.remaining,
        deletedAtM: null,
      });
      if (req.ratelimit) {
        await tx.insert(schema.ratelimits).values({
          id: newId("ratelimit"),
          workspaceId: authorizedWorkspaceId,
          name: "default",
          limit: req.ratelimit.limit,
          duration: req.ratelimit.refillRate,
          keyId: keyId,
        });
      }

      await insertUnkeyAuditLog(c, tx, {
        workspaceId: authorizedWorkspaceId,
        actor: { type: "key", id: rootKeyId },
        event: "key.create",
        description: `Created ${keyId}`,
        resources: [
          {
            type: "key",
            id: keyId,
          },
          {
            type: "keyAuth",
            // biome-ignore lint/style/noNonNullAssertion: This is legacy already no need for change
            id: api.keyAuthId!,
          },
          { type: "api", id: api.id },
        ],
        context: {
          location: c.get("location"),
          userAgent: c.get("userAgent"),
        },
      });
    });

    return c.json({
      keyId,
      key,
    });
  });
