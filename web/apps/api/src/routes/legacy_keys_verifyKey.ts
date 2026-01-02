import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { DisabledWorkspaceError, MissingRatelimitError } from "@/pkg/keys/service";
import { createRoute, z } from "@hono/zod-openapi";
import { SchemaError } from "@unkey/error";

const route = createRoute({
  deprecated: true,
  operationId: "deprecated.verifyKey",
  summary: "Verify key (deprecated)",
  "x-speakeasy-ignore": true,
  "x-excluded": true,
  method: "post",
  path: "/v1/keys/verify",
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            apiId: z
              .string()
              .optional()
              // .min(1) TODO enable after we stopped sending traffic from the agent
              .openapi({
                description: `The id of the api where the key belongs to. This is optional for now but will be required soon.
The key will be verified against the api's configuration. If the key does not belong to the api, the verification will fail.`,
                example: "api_1234",
              }),
            key: z.string().min(1).openapi({
              description: "The key to verify",
              example: "sk_1234",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The verification result",
      content: {
        "application/json": {
          schema: z.object({
            keyId: z.string().optional().openapi({
              description: "The id of the key",
              example: "key_1234",
            }),
            valid: z.boolean().openapi({
              description: `Whether the key is valid or not.
A key could be invalid for a number of reasons, for example if it has expired, has no more verifications left or if it has been deleted.`,
              example: true,
            }),
            name: z.string().optional().openapi({
              description: "The name of the key, give keys a name to easily identify their purpose",
              example: "Customer X",
            }),
            ownerId: z.string().optional().openapi({
              description:
                "The id of the tenant associated with this key. Use whatever reference you have in your system to identify the tenant. When verifying the key, we will send this field back to you, so you know who is accessing your API.",
              example: "user_123",
            }),
            meta: z
              .record(z.unknown())
              .optional()
              .openapi({
                description: "Any additional metadata you want to store with the key",
                example: {
                  roles: ["admin", "user"],
                  stripeCustomerId: "cus_1234",
                },
              }),
            createdAt: z.number().int().optional().openapi({
              description: "The unix timestamp in milliseconds when the key was created",
              example: Date.now(),
            }),

            expires: z.number().int().optional().openapi({
              description:
                "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
              example: 123,
            }),
            ratelimit: z
              .object({
                limit: z.number().int().openapi({
                  description: "Maximum number of requests that can be made inside a window",
                  example: 10,
                }),
                remaining: z.number().int().openapi({
                  description: "Remaining requests after this verification",
                  example: 9,
                }),
                reset: z
                  .number()
                  .int()
                  .openapi({
                    description: "Unix timestamp in milliseconds when the ratelimit will reset",
                    example: Date.now() + 1000 * 60 * 60,
                  }),
              })
              .optional()
              .openapi({
                description:
                  "The ratelimit configuration for this key. If this field is null or undefined, the key has no ratelimit.",
                example: {
                  limit: 10,
                  remaining: 9,
                  reset: Date.now() + 1000 * 60 * 60,
                },
              }),
            remaining: z.number().int().optional().openapi({
              description:
                "The number of requests that can be made with this key before it becomes invalid. If this field is null or undefined, the key has no request limit.",
              example: 1000,
            }),
            code: z
              .enum([
                "NOT_FOUND",
                "FORBIDDEN",
                "USAGE_EXCEEDED",
                "RATE_LIMITED",
                "UNAUTHORIZED",
                "DISABLED",
                "INSUFFICIENT_PERMISSIONS",
                "EXPIRED",
              ])
              .optional()
              .openapi({
                description: `If the key is invalid this field will be set to the reason why it is invalid.
Possible values are:
- NOT_FOUND: the key does not exist or has expired
- FORBIDDEN: the key is not allowed to access the api
- USAGE_EXCEEDED: the key has exceeded its request limit
- RATE_LIMITED: the key has been ratelimited,
- INSUFFICIENT_PERMISSIONS: you do not have the required permissions to perform this action
`,
                example: "NOT_FOUND",
              }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type LegacyKeysVerifyKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type LegacyKeysVerifyKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerLegacyKeysVerifyKey = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId, key } = c.req.valid("json");
    const { keyService } = c.get("services");

    const { val, err } = await keyService.verifyKey(c, { key, apiId });

    if (err) {
      switch (true) {
        case err instanceof SchemaError || err instanceof MissingRatelimitError:
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: err.message,
          });
        case err instanceof DisabledWorkspaceError:
          throw new UnkeyApiError({
            code: "FORBIDDEN",
            message: "workspace is disabled",
          });
      }
      throw new UnkeyApiError({
        code: "INTERNAL_SERVER_ERROR",
        message: err.message,
      });
    }

    if (!val.valid) {
      if (val.code === "NOT_FOUND" || val.code === "EXPIRED") {
        c.status(404);
      }

      return c.json({
        valid: false,
        code: val.code,
        ratelimit: val.ratelimit,
        remaining: val.remaining,
      });
    }

    return c.json({
      keyId: val.key.id,
      valid: true,
      ownerId: val.key.ownerId ?? undefined,
      meta: val.key.meta ? JSON.parse(val.key.meta) : undefined,
      expires: val.key.expires?.getTime(),
      remaining: val.remaining ?? undefined,
      ratelimit: val.ratelimit ?? undefined,
    });
  });
