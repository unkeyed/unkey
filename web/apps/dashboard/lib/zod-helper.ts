import { z } from "zod";

export const formSchema = z.object({
  bytes: z.coerce.number({
    message: "Amount must be a number and greater than 0",
  }),
  prefix: z
    .string()
    .max(16, {
      message: "Please limit the prefix to under 16 characters.",
    })
    .optional(),
  ownerId: z.string().optional(),
  name: z.string().optional(),
  metaEnabled: z.boolean(),
  meta: z.string().optional(),
  limitEnabled: z.boolean(),
  limit: z
    .object({
      remaining: z.int().positive({
        message: "Please enter a positive number",
      }),
      refill: z
        .object({
          interval: z.enum(["none", "daily", "monthly"]),
          amount: z.int().min(1).positive(),
          refillDay: z.int().min(1).max(31).optional(),
        })
        .optional(),
    })
    .optional(),
  expireEnabled: z.boolean(),
  expires: z.coerce
    .date()
    .min(new Date(Date.now() + 2 * 60000))
    .optional(),
  ratelimitEnabled: z.boolean(),
  ratelimit: z
    .object({
      type: z.enum(["consistent", "fast"]).prefault("fast"),
      refillInterval: z.coerce
        .number({
          message: "Refill interval must be greater than 0",
        })
        .positive({
          message: "Refill interval must be greater than 0",
        }),
      refillRate: z.coerce
        .number({
          message: "Refill rate must be greater than 0",
        })
        .positive({
          message: "Refill rate must be greater than 0",
        }),
      limit: z.coerce
        .number({
          message: "Refill limit must be greater than 0",
        })
        .positive({
          message: "Limit must be greater than 0",
        }),
    })
    .optional(),
});
