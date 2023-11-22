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
  })
  .openapi("Key");
