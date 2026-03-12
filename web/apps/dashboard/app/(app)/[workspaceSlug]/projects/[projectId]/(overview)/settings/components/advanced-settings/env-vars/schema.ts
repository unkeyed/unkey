import { z } from "zod";

export const envVarEntrySchema = z.object({
  id: z.string().optional(),
  environmentId: z.string().min(1, "Environment is required"),
  key: z.string().min(1, "Key is required"),
  value: z.string(),
  secret: z.boolean(),
});

export const envVarsSchema = z.object({
  envVars: z
    .array(envVarEntrySchema)
    .min(1)
    .superRefine((vars, ctx) => {
      const groups = new Map<string, number[]>();
      for (let i = 0; i < vars.length; i++) {
        const v = vars[i];
        if (!v.key) {
          continue;
        }
        const compound = `${v.environmentId}::${v.key}`;
        const indices = groups.get(compound);
        if (indices) {
          indices.push(i);
        } else {
          groups.set(compound, [i]);
        }
      }
      for (const indices of groups.values()) {
        if (indices.length > 1) {
          for (const i of indices) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: "Duplicate key in the same environment",
              path: [i, "key"],
            });
          }
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
