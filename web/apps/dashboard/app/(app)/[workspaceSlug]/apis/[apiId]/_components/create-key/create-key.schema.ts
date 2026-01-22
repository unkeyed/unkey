import { createConditionalSchema, metadataSchema } from "@/lib/schemas/metadata";
import { z } from "zod";

// Basic schemas
export const keyPrefixSchema = z
  .string()
  .max(16, {
    error: "Prefixes cannot be longer than 16 characters",
  })
  .trim()
  .refine((prefix) => !prefix.includes(" "), {
    error: "Prefixes cannot contain spaces",
  })
  .refine(
    (val) => {
      return !val || /^[a-zA-Z0-9_]+$/.test(val);
    },
    {
      error: "Prefix can only contain letters, numbers, and underscores",
    },
  )
  .optional();

export const keyBytesSchema = z.coerce
  .number()
  .int({
    error: "Key length must be a whole number (integer)",
  })
  .min(8, {
    error: "Key length is too short (minimum 8 bytes required)",
  })
  .max(255, {
    error: "Key length is too long (maximum 255 bytes allowed)",
  })
  .prefault(16);

export const nameSchema = z
  .string()
  .trim()
  .max(256, {
    error: "Name cannot exceed 256 characters",
  })
  .optional();

export const generalSchema = z.object({
  bytes: keyBytesSchema,
  prefix: keyPrefixSchema,
  externalId: z.preprocess((val) => {
    if (val === null || val === undefined || val === "") {
      return null;
    }
    if (typeof val === "string") {
      const trimmed = val.trim();
      return trimmed === "" ? null : trimmed;
    }
    return val;
  }, z.string().max(255, "External ID cannot exceed 255 characters").nullable()),
  identityId: z
    .string()
    .trim()
    .max(256, {
      error: "Identity ID cannot exceed 256 characters",
    })
    .optional()
    .nullish(),
  name: nameSchema,
  environment: z
    .string()
    .max(256, {
      error: "Environment cannot exceed 256 characters",
    })
    .trim()
    .optional(),
  enabled: z.boolean().prefault(true),
});

export const refillSchema = z.discriminatedUnion("interval", [
  z.object({
    interval: z.literal("monthly"),
    amount: z.coerce
      .number({
        error: () => "Refill amount must be a positive whole number",
      })
      .int({
        error: "Refill amount must be a whole number",
      })
      .min(1, {
        error: "Refill amount must be at least 1",
      })
      .positive({
        error: "Refill amount must be positive",
      }),
    refillDay: z.coerce
      .number({
        error: () => "Refill day must be a number between 1 and 31",
      })
      .int({
        error: "Refill day must be a whole number",
      })
      .min(1, {
        error: "Refill day must be at least 1",
      })
      .max(31, {
        error: "Refill day cannot be more than 31",
      }),
  }),
  z.object({
    interval: z.literal("daily"),
    amount: z.coerce
      .number({
        error: () => "Refill amount must be a positive whole number",
      })
      .int({
        error: "Refill amount must be a whole number",
      })
      .min(1, {
        error: "Refill amount must be at least 1",
      })
      .positive({
        error: "Refill amount must be positive",
      }),
    refillDay: z.undefined(),
  }),
  z.object({
    interval: z.literal("none").optional(),
    amount: z.undefined().optional(),
    refillDay: z.undefined().optional(),
  }),
]);

export const ratelimitItemSchema = z.object({
  id: z.string().nullish(), // Will be used only for updating case
  name: z
    .string()
    .min(3, {
      error: "Name is required and should have at least 3 characters",
    })
    .max(256, {
      error: "Name cannot exceed 256 characters",
    }),
  refillInterval: z.coerce
    .number({
      message: "Duration must be greater than 0",
    })
    .min(1000, {
      message: "Refill interval must be at least 1 second (1000ms)",
    }),
  limit: z.coerce
    .number({
      message: "Refill limit must be greater than 0",
    })
    .positive({
      message: "Limit must be greater than 0",
    }),
  autoApply: z.boolean(),
});

export const limitDataSchema = z.object({
  remaining: z.coerce
    .number({
      error: () => "Number of uses must be a positive whole number",
    })
    .int({
      error: "Number of uses must be a whole number",
    })
    .positive({
      error: "Number of uses must be positive",
    }),
  refill: refillSchema,
});

export const limitValidationSchema = z.object({
  enabled: z.literal(true),
  data: limitDataSchema,
});

export const ratelimitValidationSchema = z.object({
  enabled: z.literal(true),
  data: z.array(ratelimitItemSchema).min(1, {
    error: "At least one rate limit is required",
  }),
});

export const expirationValidationSchema = z.object({
  enabled: z.literal(true),
  data: z.preprocess(
    (val): Date | null => {
      if (val === null || val === undefined || val === "") {
        return null;
      }
      if (val instanceof Date) {
        return val;
      }
      // Only try to parse strings and numbers
      if (typeof val === "string" || typeof val === "number") {
        try {
          const date = new Date(val);
          return Number.isNaN(date.getTime()) ? null : date;
        } catch {
          return null;
        }
      }
      return null;
    },
    z
      .date({
        error: (issue) =>
          issue.input === undefined
            ? "Expiry date is required when enabled"
            : "Expiry date must be a valid date",
      })
      .refine((date) => !Number.isNaN(date.getTime()), {
        error: "Please enter a valid date",
      })
      .refine(
        (date) => {
          const minDate = new Date(Date.now() + 2 * 60000);
          return date >= minDate;
        },
        {
          error: "Expiry date must be at least 2 minutes in the future",
        },
      ),
  ),
});

// Combined schemas for forms
export const creditsSchema = z.object({
  limit: createConditionalSchema("enabled", limitValidationSchema).prefault({
    enabled: false,
    data: {
      remaining: 100,
      refill: {
        interval: "none",
      },
    },
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

export const expirationSchema = z.object({
  expiration: createConditionalSchema("enabled", expirationValidationSchema).prefault({
    enabled: false,
  }),
});

// Combined form schema for UI
export const formSchema = z
  .object({
    ...generalSchema.shape,
    ...metadataSchema.shape,
    ...creditsSchema.shape,
    ...ratelimitSchema.shape,
    ...expirationSchema.shape,
  })
  .superRefine((data, ctx) => {
    // For monthly refills, ensure refillDay is provided
    if (
      data.limit?.enabled &&
      data.limit.data?.refill?.interval === "monthly" &&
      !data.limit.data.refill.refillDay
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Refill day is required for monthly interval",
        path: ["limit", "data", "refill", "refillDay"],
      });
    }

    // Validate metadata.data field when metadata.enabled is true
    if (data.metadata?.enabled && !data.metadata.data) {
      ctx.addIssue({
        code: "custom",
        message: "Metadata is required when metadata is enabled",
        path: ["metadata", "data"],
      });
    }

    // Validate expiration.data field when expiration.enabled is true
    if (data.expiration?.enabled && !data.expiration.data) {
      ctx.addIssue({
        code: "custom",
        message: "Expiry date is required when expiration is enabled",
        path: ["expiration", "data"],
      });
    }
  });

// API/TRPC input schema
export const createKeyInputSchema = z.object({
  prefix: keyPrefixSchema,
  bytes: keyBytesSchema,
  keyAuthId: z.string(),
  externalId: z
    .string()
    .transform((s) => {
      const trimmed = s.trim();
      return trimmed === "" ? null : trimmed;
    })
    .pipe(
      z
        .string()
        .max(255, {
          error: "External ID cannot exceed 255 characters",
        })
        .nullable(),
    )
    .optional()
    .nullish(),
  identityId: z
    .string()
    .max(256, {
      error: "Identity ID cannot exceed 256 characters",
    })
    .nullish(),
  meta: z.record(z.string(), z.unknown()).optional(),
  remaining: z.int().positive().optional(),
  refill: z
    .object({
      amount: z.coerce.number().int().min(1),
      refillDay: z.int().min(1).max(31).nullable(),
    })
    .optional(),
  expires: z.int().nullish(), // unix timestamp in milliseconds
  name: nameSchema,
  ratelimit: z.array(ratelimitItemSchema).optional(),
  enabled: z.boolean().prefault(true),
  environment: z
    .string()
    .max(256, {
      error: "Environment cannot exceed 256 characters",
    })
    .optional(),
});

// Type exports
export type RatelimitItem = z.infer<typeof ratelimitItemSchema>;
export type LimitData = z.infer<typeof limitDataSchema>;
export type CreateKeyInput = z.infer<typeof createKeyInputSchema>;
export type FormValues = z.infer<typeof formSchema>;

// Helper type exports - infer directly from schemas for Zod v4 compatibility
export type RatelimitFormValues = z.infer<typeof ratelimitSchema>;
export type CreditsFormValues = z.infer<typeof creditsSchema>;
export type ExpirationFormValues = z.infer<typeof expirationSchema>;
