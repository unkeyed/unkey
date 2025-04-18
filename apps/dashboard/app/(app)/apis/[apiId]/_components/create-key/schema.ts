import { z } from "zod";

export const generalSchema = z.object({
  bytes: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type"
            ? "Key must be between 8 and 255 bytes long"
            : defaultError,
      }),
    })
    .min(8, { message: "Key must be between 8 and 255 bytes long" })
    .max(255, { message: "Key must be between 8 and 255 bytes long" })
    .default(16),
  prefix: z
    .string()
    .max(8, { message: "Prefixes cannot be longer than 8 characters" })
    .refine((prefix) => !prefix.includes(" "), {
      message: "Prefixes cannot contain spaces.",
    })
    .refine((prefix) => !prefix.endsWith("_"), {
      message:
        "Prefixes cannot end with an underscore. We'll add that automatically.",
    })
    .optional(),
  ownerId: z.string().trim().optional(),
  name: z.string().trim().optional(),
  environment: z.string().optional(),
});

export const metadataSchema = z.object({
  metaEnabled: z.boolean().default(false),
  meta: z
    .string()
    .refine(
      (s) => {
        try {
          JSON.parse(s);
          return true;
        } catch {
          return false;
        }
      },
      {
        message: "Must be valid json",
      }
    )
    .optional(),
});

export const refillSchema = z.object({
  interval: z.enum(["none", "daily", "monthly"]).default("none"),
  amount: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type"
            ? "Refill amount must be greater than 0 and a integer"
            : defaultError,
      }),
    })
    .int()
    .min(1)
    .positive()
    .optional(),
  refillDay: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type"
            ? "Refill day must be an integer between 1 and 31"
            : defaultError,
      }),
    })
    .int()
    .min(1)
    .max(31)
    .optional(),
});

export const limitSchema = z.object({
  limitEnabled: z.boolean().default(false),
  limit: z
    .object({
      remaining: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Remaining amount must be greater than 0"
                : defaultError,
          }),
        })
        .int()
        .positive({ message: "Please enter a positive number" }),
      refill: refillSchema.optional(),
    })
    .optional(),
});

export const ratelimitSchema = z.object({
  ratelimitEnabled: z.boolean().default(false),
  ratelimit: z
    .object({
      refillInterval: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Duration must be greater than 0"
                : defaultError,
          }),
        })
        .positive({ message: "Refill interval must be greater than 0" }),
      limit: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Refill limit must be greater than 0"
                : defaultError,
          }),
        })
        .positive({ message: "Limit must be greater than 0" }),
    })
    .optional(),
});

export const expirationSchema = z.object({
  expireEnabled: z.boolean().default(false),
  expires: z.coerce
    .date()
    .min(new Date(new Date().getTime() + 2 * 60000))
    .optional(),
});

// Combined form schema with validation
export const formSchema = z
  .object({
    ...generalSchema.shape,
    ...metadataSchema.shape,
    ...limitSchema.shape,
    ...ratelimitSchema.shape,
    ...expirationSchema.shape,
  })
  .superRefine((data, ctx) => {
    // Validate ratelimit fields when ratelimitEnabled is true
    if (data.ratelimitEnabled) {
      if (!data.ratelimit?.limit) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Limit is required when ratelimit is enabled",
          path: ["ratelimit", "limit"],
        });
      }

      if (!data.ratelimit?.refillInterval) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Refill interval is required when ratelimit is enabled",
          path: ["ratelimit", "refillInterval"],
        });
      }
    }

    // Validate limit fields when limitEnabled is true
    if (data.limitEnabled) {
      if (!data.limit?.remaining) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Number of uses is required when limit is enabled",
          path: ["limit", "remaining"],
        });
      }

      // If refill interval is not "none", refill amount is required
      if (
        data.limit?.refill?.interval &&
        data.limit.refill.interval !== "none"
      ) {
        if (!data.limit.refill.amount) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Refill amount is required when interval is selected",
            path: ["limit", "refill", "amount"],
          });
        }

        // If refill interval is "monthly", refill day is required
        if (
          data.limit.refill.interval === "monthly" &&
          !data.limit.refill.refillDay
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Refill day is required for monthly interval",
            path: ["limit", "refill", "refillDay"],
          });
        }
      }
    }

    // Validate meta field when metaEnabled is true
    if (data.metaEnabled && !data.meta) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Metadata is required when metadata is enabled",
        path: ["meta"],
      });
    }

    // Validate expires field when expireEnabled is true
    if (data.expireEnabled && !data.expires) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Expiry date is required when expiration is enabled",
        path: ["expires"],
      });
    }
  });

export type FormValues = z.infer<typeof formSchema>;
export type RatelimitFormValues = Pick<
  FormValues,
  "ratelimitEnabled" | "ratelimit"
>;
export type LimitFormValues = Pick<FormValues, "limitEnabled" | "limit">;
