import { z } from "zod";

export const envVarEntrySchema = z.object({
  id: z.string().optional(),
  environmentId: z.string().min(1, "Environment is required"),
  key: z
    .string()
    .min(1, "Key is required")
    .regex(/^[A-Za-z_][A-Za-z0-9_]*$/, "Must start with a letter or underscore"),
  value: z.string(),
  secret: z.boolean(),
});

export const envVarsSchema = z.object({
  envVars: z
    .array(envVarEntrySchema)
    .min(1)
    .superRefine((vars, ctx) => {
      const seen = new Map<string, number>();
      for (let i = 0; i < vars.length; i++) {
        const v = vars[i];
        if (!v.key) {
          continue;
        }
        const compound = `${v.environmentId}::${v.key}`;
        const prevIndex = seen.get(compound);
        if (prevIndex !== undefined) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Duplicate key in the same environment",
            path: [i, "key"],
          });
        } else {
          seen.set(compound, i);
        }
      }
    }),
});

export type EnvVarsFormValues = z.infer<typeof envVarsSchema>;

export function createEmptyRow(environmentId: string): EnvVarsFormValues["envVars"][number] {
  return {
    key: "",
    value: "",
    secret: false,
    environmentId,
  };
}
