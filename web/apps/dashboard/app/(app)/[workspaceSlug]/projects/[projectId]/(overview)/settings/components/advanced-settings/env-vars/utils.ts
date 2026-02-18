import type { EnvVarsFormValues } from "./schema";

export type EnvVarItem = EnvVarsFormValues["envVars"][number];

export const toTrpcType = (secret: boolean) => (secret ? "writeonly" : "recoverable");

export function computeEnvVarsDiff(original: EnvVarItem[], current: EnvVarItem[]) {
  const originalVars = original.filter((v) => v.id);
  const originalIds = new Set(originalVars.map((v) => v.id as string));
  const originalMap = new Map(originalVars.map((v) => [v.id as string, v]));

  const currentIds = new Set(current.filter((v) => v.id).map((v) => v.id as string));

  const toDelete = [...originalIds].filter((id) => !currentIds.has(id));

  const toCreate = current.filter((v) => !v.id && v.key !== "" && v.value !== "");

  const toUpdate = current.filter((v) => {
    if (!v.id) return false;
    const orig = originalMap.get(v.id);
    if (!orig) return false;
    if (v.value === "") return false;
    return v.key !== orig.key || v.value !== orig.value || v.secret !== orig.secret;
  });

  return { toDelete, toCreate, toUpdate, originalMap };
}
