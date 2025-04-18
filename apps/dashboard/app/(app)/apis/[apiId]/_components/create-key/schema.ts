import { z } from "zod";

export const generalSchema = z.object({
  bytes: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type" ? "Key must be between 8 and 255 bytes long" : defaultError,
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
      message: "Prefixes cannot end with an underscore. We'll add that automatically.",
    })
    .optional(),
  ownerId: z.string().trim().optional(),
  name: z.string().trim().optional(),
  environment: z.string().optional(),
});

export const metadataSchema = z.object({
  metadata: z
    .object({
      enabled: z.boolean().default(false),
      data: z
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
          },
        )
        .optional(),
    })
    .default({ enabled: false }),
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
  limit: z
    .object({
      enabled: z.boolean().default(false),
      data: z
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
    })
    .default({ enabled: false }),
});

export const ratelimitSchema = z.object({
  ratelimit: z
    .object({
      enabled: z.boolean().default(false),
      data: z
        .object({
          refillInterval: z.coerce
            .number({
              errorMap: (issue, { defaultError }) => ({
                message:
                  issue.code === "invalid_type" ? "Duration must be greater than 0" : defaultError,
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
    })
    .default({ enabled: false }),
});

export const expirationSchema = z.object({
  expiration: z
    .object({
      enabled: z.boolean().default(false),
      data: z.coerce
        .date()
        .min(new Date(new Date().getTime() + 2 * 60000))
        .optional(),
    })
    .default({ enabled: false }),
});

export const formSchema = z
  .object({
    ...generalSchema.shape,
    ...metadataSchema.shape,
    ...limitSchema.shape,
    ...ratelimitSchema.shape,
    ...expirationSchema.shape,
  })
  .superRefine((data, ctx) => {
    // Validate ratelimit fields when ratelimit.enabled is true
    if (data.ratelimit?.enabled) {
      // Default values should be set when enabling
      const limit = data.ratelimit.data?.limit;
      const refillInterval = data.ratelimit.data?.refillInterval;

      if (!limit || limit <= 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Limit is required when ratelimit is enabled",
          path: ["ratelimit", "data", "limit"],
        });
      }

      if (!refillInterval || refillInterval <= 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Refill interval is required when ratelimit is enabled",
          path: ["ratelimit", "data", "refillInterval"],
        });
      }
    }

    // Validate limit fields when limit.enabled is true
    if (data.limit?.enabled) {
      const remaining = data.limit.data?.remaining;

      if (!remaining || remaining <= 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Number of uses is required when limit is enabled",
          path: ["limit", "data", "remaining"],
        });
      }

      // If refill interval is not "none", refill amount is required
      if (data.limit.data?.refill?.interval && data.limit.data.refill.interval !== "none") {
        if (!data.limit.data.refill.amount) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Refill amount is required when interval is selected",
            path: ["limit", "data", "refill", "amount"],
          });
        }

        // If refill interval is "monthly", refill day is required
        if (data.limit.data.refill.interval === "monthly" && !data.limit.data.refill.refillDay) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Refill day is required for monthly interval",
            path: ["limit", "data", "refill", "refillDay"],
          });
        }
      }
    }

    // Validate metadata.data field when metadata.enabled is true
    if (data.metadata?.enabled && !data.metadata.data) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Metadata is required when metadata is enabled",
        path: ["metadata", "data"],
      });
    }

    // Validate expiration.data field when expiration.enabled is true
    if (data.expiration?.enabled && !data.expiration.data) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Expiry date is required when expiration is enabled",
        path: ["expiration", "data"],
      });
    }
  });

export type FormValues = z.infer<typeof formSchema>;
export type RatelimitFormValues = Pick<FormValues, "ratelimit">;
export type LimitFormValues = Pick<FormValues, "limit">;
export type MetadataFormValues = Pick<FormValues, "metadata">;
export type ExpirationFormValues = Pick<FormValues, "expiration">;
