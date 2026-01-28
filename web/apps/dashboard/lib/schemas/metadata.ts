import { z } from "zod";

/**
 * Creates a conditional schema that validates fields only when enabled is true.
 * Uses Zod v4 discriminated unions for proper type inference with react-hook-form.
 *
 * This function leverages Zod v4's discriminated union feature, which is specifically
 * designed for conditional validation scenarios. The discriminator field (typically "enabled")
 * determines which branch of the union is validated, providing superior type safety and
 * compatibility with form libraries like react-hook-form compared to loose object unions.
 *
 * The disabled branch uses passthrough() to allow additional properties (like default values)
 * without validation, while the enabled branch enforces strict validation.
 *
 * @param enabledPath - The path to the boolean field that controls validation (default: "enabled")
 * @param schema - The schema to apply when enabled is true (must include the enabledPath field set to z.literal(true))
 * @returns A discriminated union schema compatible with zodResolver
 *
 * @example
 * ```typescript
 * const conditionalSchema = createConditionalSchema("enabled", z.object({
 *   enabled: z.literal(true),
 *   data: z.string(),
 * }));
 * ```
 */
export const createConditionalSchema = <
  T extends z.ZodRawShape,
  EnabledPath extends string = "enabled",
>(
  enabledPath: EnabledPath,
  schema: z.ZodObject<T>,
) => {
  return z.discriminatedUnion(enabledPath, [
    // When enabled is false, allow additional properties without validation
    // This enables default values to be set via prefault()
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
