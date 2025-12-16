import { z } from "zod";
import { createConditionalSchema } from "./metadata";

export const ratelimitItemSchema = z.object({
  id: z.string().nullish(), // Will be used only for updating case
  name: z
    .string()
    .min(3, { message: "Name is required and should have at least 3 characters" })
    .max(256, { message: "Name cannot exceed 256 characters" }),
  refillInterval: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message: issue.code === "invalid_type" ? "Duration must be a valid number" : defaultError,
      }),
    })
    .min(1000, { message: "Refill interval must be at least 1 second (1000ms)" }),
  limit: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type" ? "Limit must be a valid number" : defaultError,
      }),
    })
    .positive({ message: "Limit must be greater than 0" }),
  autoApply: z.boolean(),
});

export const ratelimitValidationSchema = z.object({
  enabled: z.literal(true),
  data: z
    .array(ratelimitItemSchema)
    .min(1, { message: "At least one rate limit is required" })
    .superRefine((items, ctx) => {
      const seenNames = new Set<string>();
      for (let i = 0; i < items.length; i++) {
        const name = items[i].name;
        if (seenNames.has(name)) {
          ctx.addIssue({
            code: "custom",
            message: "Ratelimit name must be unique",
            path: ["data", i, "name"],
          });
        }
        seenNames.add(name);
      }
    }),
});

export const ratelimitSchema = z.object({
  ratelimit: createConditionalSchema("enabled", ratelimitValidationSchema).default({
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

// Manual type for form context to avoid conditional schema union issues
export type RatelimitFormContextValues = {
  ratelimit: {
    enabled: boolean;
    data: RatelimitItem[];
  };
};
