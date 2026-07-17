import { z } from "zod";

const IDENTITY_METADATA_SIZE_BYTES_MAX = 1024 * 1024;
const KEY_METADATA_SIZE_BYTES_MAX = 65_535;

function createMetadataValueSchema(sizeBytesMax: number, sizeError: string) {
  return z
    .record(z.string(), z.unknown())
    .refine((metadata) => Object.keys(metadata).length <= 100, {
      error: "Metadata cannot contain more than 100 properties",
    })
    .refine(
      (metadata) => new TextEncoder().encode(JSON.stringify(metadata)).length <= sizeBytesMax,
      { error: sizeError },
    );
}

const keyMetadataValueSchema = createMetadataValueSchema(
  KEY_METADATA_SIZE_BYTES_MAX,
  "Metadata cannot exceed 65,535 bytes",
);
const identityMetadataValueSchema = createMetadataValueSchema(
  IDENTITY_METADATA_SIZE_BYTES_MAX,
  "Metadata cannot exceed 1 MiB",
);

export function parseIdentityMetadata(value: string): Record<string, unknown> {
  const parsed: unknown = JSON.parse(value);
  return identityMetadataValueSchema.parse(parsed);
}

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

function createMetadataValidationSchema(metadataValueSchema: typeof keyMetadataValueSchema) {
  return z.object({
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
      .superRefine((value, context) => {
        let parsed: unknown;
        try {
          parsed = JSON.parse(value);
        } catch {
          context.addIssue({ code: "custom", message: "Must be valid JSON" });
          return;
        }

        const result = metadataValueSchema.safeParse(parsed);
        if (!result.success) {
          for (const issue of result.error.issues) {
            context.addIssue({ code: "custom", message: issue.message });
          }
        }
      }),
  });
}

export const metadataValidationSchema = createMetadataValidationSchema(keyMetadataValueSchema);
export const identityMetadataValidationSchema = createMetadataValidationSchema(
  identityMetadataValueSchema,
);

export const metadataSchema = z.object({
  metadata: createConditionalSchema("enabled", metadataValidationSchema).prefault({
    enabled: false,
  }),
});

export const identityMetadataSchema = z.object({
  metadata: createConditionalSchema("enabled", identityMetadataValidationSchema).prefault({
    enabled: false,
  }),
});

export type MetadataFormValues = z.infer<typeof metadataSchema>;
