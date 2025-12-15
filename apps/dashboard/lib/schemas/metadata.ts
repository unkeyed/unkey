import { z } from "zod";

// Helper function for creating conditional schemas based on the "enabled" flag
export const createConditionalSchema = <
  T extends z.ZodRawShape,
  U extends z.UnknownKeysParam = z.UnknownKeysParam,
  V extends z.ZodTypeAny = z.ZodTypeAny,
  EnabledPath extends string = "enabled",
>(
  enabledPath: EnabledPath,
  schema: z.ZodObject<T, U, V>,
) => {
  return z.union([
    // When enabled is false, don't validate other fields
    z
      .object({
        [enabledPath]: z.literal(false),
      } as { [K in EnabledPath]: z.ZodLiteral<false> })
      .passthrough(),
    // When enabled is true, apply all validations
    schema,
  ]);
};

export const metadataValidationSchema = z.object({
  enabled: z.literal(true),
  data: z
    .string({
      required_error: "Metadata is required",
      invalid_type_error: "Metadata must be a JSON",
    })
    .trim()
    .min(2, { message: "Metadata must contain valid JSON" })
    .max(65534, {
      message: "Metadata cannot exceed 65535 characters (text field limit)",
    })
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
        message: "Must be valid JSON",
      },
    ),
});

export const metadataSchema = z.object({
  metadata: createConditionalSchema("enabled", metadataValidationSchema).default({
    enabled: false,
  }),
});

export type MetadataFormValues = z.infer<typeof metadataSchema>;
