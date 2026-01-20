import { z } from "zod";

// Helper function for creating conditional schemas based on the "enabled" flag
export const createConditionalSchema = <
  T extends z.ZodRawShape,
  EnabledPath extends string = "enabled",
>(
  enabledPath: EnabledPath,
  schema: z.ZodObject<T>,
) => {
  return z.union([
    // When enabled is false, don't validate other fields
    z.looseObject({
      [enabledPath]: z.literal(false),
    } as { [K in EnabledPath]: z.ZodLiteral<false> }),
    // When enabled is true, apply all validations
    schema,
  ]);
};

export const metadataValidationSchema = z.object({
  enabled: z.literal(true),
  data: z
    .string({
      error: (issue) =>
        issue.input === undefined ? "Metadata is required" : "Metadata must be a JSON",
    })
    .trim()
    .min(2, {
      error: "Metadata must contain valid JSON",
    })
    .max(65534, {
      error: "Metadata cannot exceed 65535 characters (text field limit)",
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
        error: "Must be valid JSON",
      },
    ),
});

export const metadataSchema = z.object({
  metadata: createConditionalSchema("enabled", metadataValidationSchema).prefault({
    enabled: false,
  }),
});

export type MetadataFormValues = z.infer<typeof metadataSchema>;
