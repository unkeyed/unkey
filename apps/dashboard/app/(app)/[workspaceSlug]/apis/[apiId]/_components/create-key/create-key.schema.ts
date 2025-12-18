import { createConditionalSchema, metadataSchema } from "@/lib/schemas/metadata";
import { z } from "zod";

// Basic schemas
export const keyPrefixSchema = z
  .string()
  .max(16, { message: "Prefixes cannot be longer than 16 characters" })
  .trim()
  .refine((prefix) => !prefix.includes(" "), {
    message: "Prefixes cannot contain spaces",
  })
  .refine(
    (val) => {
      return !val || /^[a-zA-Z0-9_]+$/.test(val);
    },
    { message: "Prefix can only contain letters, numbers, and underscores" },
  )
  .optional();

export const keyBytesSchema = z.coerce
  .number()
  .int({ message: "Key length must be a whole number (integer)" })
  .min(8, { message: "Key length is too short (minimum 8 bytes required)" })
  .max(255, { message: "Key length is too long (maximum 255 bytes allowed)" })
  .default(16);

export const nameSchema = z
  .string()
  .trim()
  .max(256, { message: "Name cannot exceed 256 characters" })
  .optional();

export const generalSchema = z.object({
  bytes: keyBytesSchema,
  prefix: keyPrefixSchema,
  externalId: z
    .string()
    .transform((s) => s.trim())
    .refine((trimmed) => trimmed.length >= 1, "External ID must be at least 1 character")
    .refine((trimmed) => trimmed.length <= 255, "External ID cannot exceed 255 characters")
    .refine((trimmed) => trimmed !== "", "External ID cannot be only whitespace")
    .optional()
    .nullish(),
  identityId: z
    .string()
    .trim()
    .max(256, { message: "Identity ID cannot exceed 256 characters" })
    .optional()
    .nullish(),
  name: nameSchema,
  environment: z
    .string()
    .max(256, { message: "Environment cannot exceed 256 characters" })
    .trim()
    .optional(),
  enabled: z.boolean().default(true),
});

export const refillSchema = z.discriminatedUnion("interval", [
  z.object({
    interval: z.literal("monthly"),
    amount: z.coerce
      .number({
        errorMap: () => ({
          message: "Refill amount must be a positive whole number",
        }),
      })
      .int({ message: "Refill amount must be a whole number" })
      .min(1, { message: "Refill amount must be at least 1" })
      .positive({ message: "Refill amount must be positive" }),
    refillDay: z.coerce
      .number({
        errorMap: () => ({
          message: "Refill day must be a number between 1 and 31",
        }),
      })
      .int({ message: "Refill day must be a whole number" })
      .min(1, { message: "Refill day must be at least 1" })
      .max(31, { message: "Refill day cannot be more than 31" }),
  }),
  z.object({
    interval: z.literal("daily"),
    amount: z.coerce
      .number({
        errorMap: () => ({
          message: "Refill amount must be a positive whole number",
        }),
      })
      .int({ message: "Refill amount must be a whole number" })
      .min(1, { message: "Refill amount must be at least 1" })
      .positive({ message: "Refill amount must be positive" }),
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
    .min(3, { message: "Name is required and should have at least 3 characters" })
    .max(256, { message: "Name cannot exceed 256 characters" }),
  refillInterval: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message: issue.code === "invalid_type" ? "Duration must be greater than 0" : defaultError,
      }),
    })
    .min(1000, { message: "Refill interval must be at least 1 second (1000ms)" }),
  limit: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type" ? "Refill limit must be greater than 0" : defaultError,
      }),
    })
    .positive({ message: "Limit must be greater than 0" }),
  autoApply: z.boolean(),
});

export const limitDataSchema = z.object({
  remaining: z.coerce
    .number({
      errorMap: () => ({
        message: "Number of uses must be a positive whole number",
      }),
    })
    .int({ message: "Number of uses must be a whole number" })
    .positive({ message: "Number of uses must be positive" }),
  refill: refillSchema,
});

export const limitValidationSchema = z.object({
  enabled: z.literal(true),
  data: limitDataSchema,
});

export const ratelimitValidationSchema = z.object({
  enabled: z.literal(true),
  data: z.array(ratelimitItemSchema).min(1, { message: "At least one rate limit is required" }),
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
        required_error: "Expiry date is required when enabled",
        invalid_type_error: "Expiry date must be a valid date",
      })
      .refine((date) => !Number.isNaN(date.getTime()), {
        message: "Please enter a valid date",
      })
      .refine(
        (date) => {
          const minDate = new Date(Date.now() + 2 * 60000);
          return date >= minDate;
        },
        {
          message: "Expiry date must be at least 2 minutes in the future",
        },
      ),
  ),
});

// Combined schemas for forms
export const creditsSchema = z.object({
  limit: createConditionalSchema("enabled", limitValidationSchema)
    .optional()
    .default({
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

export const expirationSchema = z.object({
  expiration: createConditionalSchema("enabled", expirationValidationSchema).default({
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
        code: z.ZodIssueCode.custom,
        message: "Refill day is required for monthly interval",
        path: ["limit", "data", "refill", "refillDay"],
      });
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
        .max(255, { message: "External ID cannot exceed 255 characters" })
        .nullable(),
    )
    .optional()
    .nullish(),
  identityId: z
    .string()
    .max(256, { message: "Identity ID cannot exceed 256 characters" })
    .nullish(),
  meta: z.record(z.unknown()).optional(),
  remaining: z.number().int().positive().optional(),
  refill: z
    .object({
      amount: z.coerce.number().int().min(1),
      refillDay: z.number().int().min(1).max(31).nullable(),
    })
    .optional(),
  expires: z.number().int().nullish(), // unix timestamp in milliseconds
  name: nameSchema,
  ratelimit: z.array(ratelimitItemSchema).optional(),
  enabled: z.boolean().default(true),
  environment: z
    .string()
    .max(256, { message: "Environment cannot exceed 256 characters" })
    .optional(),
});

// Type exports
export type RatelimitItem = z.infer<typeof ratelimitItemSchema>;
export type LimitData = z.infer<typeof limitDataSchema>;
export type CreateKeyInput = z.infer<typeof createKeyInputSchema>;
export type FormValues = z.infer<typeof formSchema>;

export type FormValueTypes = {
  bytes: number;
  prefix?: string;
  ownerId?: string;
  name?: string;
  environment?: string;
  metadata: {
    enabled: boolean;
    data?: string;
  };
  limit?: {
    enabled: boolean;
    data?: LimitData;
  };
  ratelimit: {
    enabled: boolean;
    data: RatelimitItem[];
  };
  expiration: {
    enabled: boolean;
    data?: Date;
  };
};

// Helper type exports
export type RatelimitFormValues = Pick<FormValueTypes, "ratelimit">;
export type CreditsFormValues = Pick<FormValueTypes, "limit">;
export type ExpirationFormValues = Pick<FormValueTypes, "expiration">;
