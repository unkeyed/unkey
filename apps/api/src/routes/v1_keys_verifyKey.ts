import { UnkeyApiError, openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";
import { permissionQuerySchema } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "v1.keys.verifyKey",
  method: "post",
  path: "/v1/keys.verifyKey",
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z
            .object({
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
              authorization: z
                .object({
                  permissions: z.any(permissionQuerySchema).openapi({
                    type: "object",
                    description: "A query for which permissions you require",
                    example: {
                      or: [{ and: ["dns.record.read", "dns.record.update"] }, "admin"],
                    },
                  }),
                })
                .optional()
                .openapi({
                  description: "Perform RBAC checks",
                }),
              ratelimit: z
                .object({
                  cost: z.number().int().positive().optional().default(1).openapi({
                    description:
                      "Override how many tokens are deducted during the ratelimit operation.",
                  }),
                })
                .optional(),
            })
            .openapi("V1KeysVerifyKeyRequest"),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The verification result",
      content: {
        "application/json": {
          schema: z
            .object({
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
                description:
                  "The name of the key, give keys a name to easily identifiy their purpose",
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
              expires: z.number().optional().openapi({
                description:
                  "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
                example: 123,
              }),
              ratelimit: z
                .object({
                  limit: z.number().openapi({
                    description: "Maximum number of requests that can be made inside a window",
                    example: 10,
                  }),
                  remaining: z.number().openapi({
                    description: "Remaining requests after this verification",
                    example: 9,
                  }),
                  reset: z.number().openapi({
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
              remaining: z.number().optional().openapi({
                description:
                  "The number of requests that can be made with this key before it becomes invalid. If this field is null or undefined, the key has no request limit.",
                example: 1000,
              }),
              code: z
                .enum([
                  "VALID",
                  "NOT_FOUND",
                  "FORBIDDEN",
                  "USAGE_EXCEEDED",
                  "RATE_LIMITED",
                  "UNAUTHORIZED",
                  "DISABLED",
                  "INSUFFICIENT_PERMISSIONS",
                ])
                .openapi({
                  description: `A machine readable code why the key is not valid.
Possible values are:
- VALID: the key is valid and you should proceed
- NOT_FOUND: the key does not exist or has expired
- FORBIDDEN: the key is not allowed to access the api
- USAGE_EXCEEDED: the key has exceeded its request limit
- RATE_LIMITED: the key has been ratelimited
- UNAUTHORIZED: the key is not authorized
- DISABLED: the key is disabled
- INSUFFICIENT_PERMISSIONS: you do not have the required permissions to perform this action
`,
                }),
              enabled: z.boolean().optional().openapi({
                description:
                  "Sets the key to be enabled or disabled. Disabled keys will not verify.",
              }),
              permissions: z
                .array(z.string())
                .optional()
                .openapi({
                  description: "A list of all the permissions this key is connected to.",
                  example: ["dns.record.update", "dns.record.delete"],
                }),
              environment: z.string().optional().openapi({
                description:
                  "The environment of the key, this is what what you set when you crated the key",
                example: "test",
              }),
            })
            .openapi("V1KeysVerifyKeyResponse"),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type V1KeysVerifyKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysVerifyKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysVerifyKey = (app: App) =>
  app.openapi(route, async (c) => {
    const { apiId, key, authorization, ratelimit } = c.req.valid("json");
    const { keyService } = c.get("services");

    const { val, err } = await keyService.verifyKey(c, {
      key,
      apiId,
      permissionQuery: authorization?.permissions,
      ratelimit: ratelimit,
    });
    if (err) {
      switch (err.name) {
        case "SchemaError":
          throw new UnkeyApiError({
            code: "BAD_REQUEST",
            message: err.message,
          });
        case "DisabledWorkspaceError":
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
      return c.json({
        keyId: val.key?.id,
        valid: false,
        code: val.code,
        rateLimit: val.ratelimit,
        remaining: val.remaining,
      });
    }

    return c.json({
      keyId: val.key.id,
      valid: true,
      name: val.key.name ?? undefined,
      ownerId: val.key.ownerId ?? undefined,
      meta: val.key.meta ? JSON.parse(val.key.meta) : undefined,
      expires: val.key.expires?.getTime(),
      remaining: val.remaining ?? undefined,
      ratelimit: val.ratelimit ?? undefined,
      enabled: val.key.enabled,
      permissions: val.permissions,
      environment: val.key.environment ?? undefined,
      code: "VALID" as const,
    });
  });
