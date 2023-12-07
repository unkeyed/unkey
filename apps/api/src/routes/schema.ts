import { z } from "zod";

export const keySchema = z
  .object({
    id: z.string().openapi({
      description: "The id of the key",
      example: "key_1234",
    }),
    start: z.string().openapi({
      description: "The first few characters of the key to visually identify it",
      example: "sk_5j1",
    }),
    workspaceId: z.string().openapi({
      description: "The id of the workspace that owns the key",
      example: "ws_1234",
    }),
    apiId: z.string().optional().openapi({
      description: "The id of the api that this key is for",
      example: "api_1234",
    }),
    name: z.string().optional().openapi({
      description: "The name of the key, give keys a name to easily identifiy their purpose",
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
    createdAt: z.number().openapi({
      description: "The unix timestamp in milliseconds when the key was created",
      example: Date.now(),
    }),
    deletedAt: z.number().optional().openapi({
      description:
        "The unix timestamp in milliseconds when the key was deleted. We don't delete the key outright, you can restore it later.",
      example: Date.now(),
    }),
    expires: z.number().optional().openapi({
      description:
        "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
      example: Date.now(),
    }),
    remaining: z.number().optional().openapi({
      description:
        "The number of requests that can be made with this key before it becomes invalid. If this field is null or undefined, the key has no request limit.",
      example: 1000,
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
          description: "Determines the speed at which tokens are refilled, in milliseconds.",
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
    refill: z
      .object({
        refillInterval: z.enum(["daily", "monthly"]).openapi({
          description: "Determines the rate at kills will be refilled.",
          example: "daily",
        }),
        refillIncrement: z.number().int().openapi({
          description: "Sets how many uses to refill during each refillInterval.",
          example: 100,
        }),
      })
      .optional(),
  })
  .openapi("Key");
