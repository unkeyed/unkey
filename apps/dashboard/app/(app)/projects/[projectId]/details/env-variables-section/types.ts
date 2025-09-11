import { z } from "zod";

export const EnvVarTypeSchema = z.enum(["env", "secret"]);
export type EnvVarType = z.infer<typeof EnvVarTypeSchema>;

export const envVarSchema = z
  .object({
    id: z.string(),
    key: z.string(),
    value: z.string(), // For 'env': actual value, for 'secret': encrypted blob
    type: EnvVarTypeSchema,
    decryptedValue: z.string().optional(), // Only populated when user views secret
  })
  .superRefine((data, ctx) => {
    if (data.type !== "secret" && data.decryptedValue) {
      ctx.addIssue({
        code: "custom",
        path: ["decryptedValue"],
        message: "Only allowed for secrets",
      });
    }
  });

export type EnvVar = z.infer<typeof envVarSchema>;

export const EnvVarFormSchema = z.object({
  key: z
    .string()
    .trim()
    .min(1, "Variable name is required")
    .regex(/^[A-Za-z][A-Za-z0-9_]*$/, "Use letters, numbers, and underscores; start with a letter"),
  value: z.string().min(1, "Variable value is required"),
  type: EnvVarTypeSchema,
});
export type EnvVarFormData = z.infer<typeof EnvVarFormSchema>;

export const EnvironmentSchema = z.enum(["production", "preview"]);
export type Environment = z.infer<typeof EnvironmentSchema>;
