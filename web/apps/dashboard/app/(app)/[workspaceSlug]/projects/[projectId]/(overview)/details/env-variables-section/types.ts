import { z } from "zod";

// Regex for valid environment variable keys: must start with uppercase letter,
// followed by uppercase letters, numbers, or underscores
export const ENV_VAR_KEY_REGEX = /^[A-Z][A-Z0-9_]*$/;

// Both types are encrypted in the database
// - recoverable: can be decrypted and shown in the UI
// - writeonly: cannot be read back after creation
export const EnvVarTypeSchema = z.enum(["recoverable", "writeonly"]);
export type EnvVarType = z.infer<typeof EnvVarTypeSchema>;

export const envVarSchema = z.object({
  id: z.string(),
  key: z.string(),
  value: z.string(), // For 'recoverable': decrypted value, for 'writeonly': masked (e.g. "••••••••")
  type: EnvVarTypeSchema,
  description: z.string().nullable().optional(),
});

export type EnvVar = z.infer<typeof envVarSchema>;

export const EnvVarFormSchema = z.object({
  key: z
    .string()
    .trim()
    .min(1, "Variable name is required")
    .regex(ENV_VAR_KEY_REGEX, "Must be UPPERCASE with letters, numbers, and underscores only"),
  value: z.string().min(1, "Variable value is required"),
  type: EnvVarTypeSchema,
});
export type EnvVarFormData = z.infer<typeof EnvVarFormSchema>;

// Environment slug type - this should match the slugs stored in the database
export type Environment = string;
