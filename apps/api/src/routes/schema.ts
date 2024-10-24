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
    createdAt: z.number().int().openapi({
      description: "The unix timestamp in milliseconds when the key was created",
      example: Date.now(),
    }),
    updatedAt: z.number().int().optional().openapi({
      description: "The unix timestamp in milliseconds when the key was last updated",
      example: Date.now(),
    }),

    expires: z.number().int().optional().openapi({
      description:
        "The unix timestamp in milliseconds when the key will expire. If this field is null or undefined, the key is not expiring.",
      example: Date.now(),
    }),
    remaining: z.number().int().optional().openapi({
      description:
        "The number of requests that can be made with this key before it becomes invalid. If this field is null or undefined, the key has no request limit.",
      example: 1000,
    }),
    refill: z
      .object({
        interval: z.enum(["daily", "monthly"]).openapi({
          description:
            "Determines the rate at which verifications will be refilled. When 'daily' is set for 'interval' 'refillDay' will be set to null.",
          example: "daily",
        }),
        amount: z.number().int().openapi({
          description: "Resets `remaining` to this value every interval.",
          example: 100,
        }),
        refillDay: z.number().min(1).max(31).default(1).nullable().openapi({
          description:
            "The day verifications will refill each month, when interval is set to 'monthly'. Value is not zero-indexed making 1 the first day of the month. If left blank it will default to the first day of the month. When 'daily' is set for 'interval' 'refillDay' will be set to null.",
          example: 15,
        }),
        lastRefillAt: z.number().int().optional().openapi({
          description: "The unix timestamp in miliseconds when the key was last refilled.",
          example: 100,
        }),
      })
      .optional()
      .openapi({
        description:
          "Unkey allows you to refill remaining verifications on a key on a regular interval.",
        example: {
          interval: "monthly",
          amount: 10,
          refillDay: 10,
        },
      }),

    ratelimit: z
      .object({
        async: z.boolean().openapi({
          description: "",
        }),
        type: z
          .enum(["fast", "consistent"])
          .optional()
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
        refillRate: z.number().int().min(1).optional().openapi({
          description: "How many tokens to refill during each refillInterval.",
        }),
        refillInterval: z.number().int().min(1).optional().openapi({
          description: "Determines the speed at which tokens are refilled, in milliseconds.",
        }),
        duration: z.number().int().min(1).openapi({
          description: "The duration of the ratelimit window, in milliseconds.",
        }),
      })
      .optional()
      .openapi({
        description: "Unkey comes with per-key ratelimiting out of the box.",
        example: {
          async: true,
          limit: 10,
          duration: 60,
        },
      }),
    roles: z
      .array(z.string())
      .optional()
      .openapi({
        description: "All roles this key belongs to",
        example: ["admin", "finance"],
      }),
    permissions: z
      .array(z.string())
      .optional()
      .openapi({
        description: "All permissions this key has",
        example: ["domain.dns.create_record", "finance.read_receipt"],
      }),
    enabled: z.boolean().optional().openapi({
      description: "Sets if key is enabled or disabled. Disabled keys are not valid.",
      example: true,
    }),
    plaintext: z.string().optional().openapi({
      description: "The key in plaintext",
    }),
    identity: z
      .object({
        id: z.string().openapi({
          description: "The id of the identity",
        }),
        externalId: z.string().openapi({
          description: "The external id of the identity",
        }),
        meta: z.record(z.unknown()).optional().openapi({
          description: "Any additional metadata attached to the identity",
        }),
      })
      .optional()
      .openapi({
        description: "The identity of the key",
      }),
  })
  .openapi("Key");
