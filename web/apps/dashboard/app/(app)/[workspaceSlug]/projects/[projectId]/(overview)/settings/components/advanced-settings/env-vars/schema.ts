import { z } from "zod";

export const envVarEntrySchema = z.object({
  id: z.string().optional(),
  key: z
    .string()
    .min(1, "Key is required")
    .regex(/^[A-Za-z_][A-Za-z0-9_]*$/, "Must start with a letter or underscore"),
  value: z.string(),
  secret: z.boolean(),
});

export const envVarsSchema = z.object({
  envVars: z.array(envVarEntrySchema).min(1),
});

export type EnvVarsFormValues = z.infer<typeof envVarsSchema>;

export const EMPTY_ROW: EnvVarsFormValues["envVars"][number] = {
  key: "",
  value: "",
  secret: false,
};
