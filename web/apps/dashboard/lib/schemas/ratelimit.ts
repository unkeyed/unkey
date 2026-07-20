import { z } from "zod";
import { createConditionalSchema } from "./metadata";

export const ratelimitItemSchema = z.object({
  id: z.string().nullish(), // Will be used only for updating case
  name: z
    .string()
    .min(3, {
      error: "Name is required and should have at least 3 characters",
    })
    .max(128, {
      message: "Name cannot exceed 128 characters",
    }),
  refillInterval: z.coerce
    .number({
      message: "Duration must be a valid number",
    })
    .int({ message: "Refill interval must be a whole number" })
    .min(1000, {
      message: "Refill interval must be at least 1 second (1000ms)",
    }),
  limit: z.coerce
    .number({
      message: "Limit must be a valid number",
    })
    .int({ message: "Limit must be a whole number" })
    .positive({
      message: "Limit must be greater than 0",
    }),
  autoApply: z.boolean().default(false),
});

export const ratelimitValidationSchema = z.object({
  enabled: z.literal(true),
  data: z
    .array(ratelimitItemSchema)
    .min(1, {
      error: "At least one rate limit is required",
    })
    .max(50, {
      error: "An identity cannot have more than 50 rate limits",
    })
    .superRefine((items, ctx) => {
      const seenNames = new Set<string>();
      for (let i = 0; i < items.length; i++) {
        const name = items[i].name;
        if (seenNames.has(name)) {
          ctx.addIssue({
            code: "custom",
            message: "Ratelimit name must be unique",
            path: [i, "name"],
          });
        }
        seenNames.add(name);
      }
    }),
});

export const ratelimitSchema = z.object({
  ratelimit: createConditionalSchema("enabled", ratelimitValidationSchema).prefault({
    enabled: false,
    data: [
      {
        name: "default",
        limit: 10,
        refillInterval: 1000,
        autoApply: true,
      },
    ],
  }),
});

// Type exports
export type RatelimitItem = z.infer<typeof ratelimitItemSchema>;
export type RatelimitFormValues = z.infer<typeof ratelimitSchema>;
