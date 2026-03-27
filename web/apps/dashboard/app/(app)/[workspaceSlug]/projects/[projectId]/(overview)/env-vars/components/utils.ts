import type { EnvVarsFormValues } from "./schema";

export type EnvVarEntry = EnvVarsFormValues["envVars"][number];

export type FlatEnvVarRecord = {
  key: string;
  value: string;
  secret: boolean;
  description: string | undefined;
  environmentId: string;
};

export const toTrpcType = (secret: boolean) => (secret ? "writeonly" : "recoverable");

export function expandToFlatRecords(
  entries: EnvVarEntry[],
  environmentIds: string[],
  secret: boolean,
): FlatEnvVarRecord[] {
  return entries.flatMap((entry) =>
    environmentIds.map((envId) => ({
      key: entry.key,
      value: entry.value,
      secret,
      description: entry.description,
      environmentId: envId,
    })),
  );
}