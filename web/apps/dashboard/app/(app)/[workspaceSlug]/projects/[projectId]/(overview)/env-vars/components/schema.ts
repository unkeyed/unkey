import { z } from "zod";

export const envVarEntrySchema = z.object({
  key: z.string().min(1, "Variable name is required"),
  value: z.string(),
  description: z.string().optional(),
});

export const envVarsSchema = z.object({
  envVars: z
    .array(envVarEntrySchema)
    .min(1)
    .superRefine((vars, ctx) => {
      const seen = new Map<string, number[]>();
      for (let i = 0; i < vars.length; i++) {
        if (!vars[i].key) {
          continue;
        }
        const indices = seen.get(vars[i].key);
        if (indices) {
          indices.push(i);
        } else {
          seen.set(vars[i].key, [i]);
        }
      }
      for (const indices of seen.values()) {
        if (indices.length > 1) {
          for (const i of indices) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: "Duplicate variable name",
              path: [i, "key"],
            });
          }
        }
      }
    }),
  environmentId: z.string().min(1, "Environment is required"),
  secret: z.boolean(),
});

export type EnvVarsFormValues = z.infer<typeof envVarsSchema>;

export function createEmptyEntry(): EnvVarsFormValues["envVars"][number] {
  return {
    key: "",
    value: "",
    description: "",
  };
}

type ExistingEnvVar = { key: string; environmentId: string };

/**
 * Returns indices of form entries whose key already exists in the given environment(s).
 */
export function findConflicts(
  entries: { key: string }[],
  environmentId: string,
  existingVars: ExistingEnvVar[],
  allEnvironmentIds: string[],
): number[] {
  const targetEnvIds = environmentId === "__all__" ? allEnvironmentIds : [environmentId];
  const existingSet = new Set(existingVars.map((v) => `${v.key}\0${v.environmentId}`));

  const conflictIndices: number[] = [];
  for (let i = 0; i < entries.length; i++) {
    if (!entries[i].key) {
      continue;
    }
    for (const envId of targetEnvIds) {
      if (existingSet.has(`${entries[i].key}\0${envId}`)) {
        conflictIndices.push(i);
        break;
      }
    }
  }
  return conflictIndices;
}
