import type { UnkeyAuditLog } from "@/lib/audit";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { z } from "zod";

const MAX_VARIABLES = 200;

export const addEnvVarsInput = z.object({
  appId: z.string().min(1),
  environmentIds: z.array(z.string().min(1)).min(1),
  variables: z
    .array(
      z.object({
        key: envVarKeySchema,
        value: envVarValueSchema,
        kind: z.enum(["recoverable", "writeonly"]),
      }),
    )
    .min(1)
    .max(MAX_VARIABLES, `Add at most ${MAX_VARIABLES} variables at once`)
    .refine(
      (variables) => new Set(variables.map((v) => v.key)).size === variables.length,
      "Each key may appear only once",
    ),
});

export type AddEnvVarsInput = z.infer<typeof addEnvVarsInput>;
type Variable = AddEnvVarsInput["variables"][number];

export type AddEnvVarsResult =
  | { status: "taken"; keys: string[] }
  | { status: "added"; count: number };

export function takenKeys(existing: { key: string }[]): string[] {
  return [...new Set(existing.map((row) => row.key))].sort();
}

export type EncryptedVariable = Variable & { id: string; ciphertext: string };

export function envVarRows({
  workspaceId,
  appId,
  environmentId,
  variables,
}: {
  workspaceId: string;
  appId: string;
  environmentId: string;
  variables: EncryptedVariable[];
}) {
  return variables.map((v) => ({
    id: v.id,
    workspaceId,
    appId,
    environmentId,
    key: v.key,
    value: v.ciphertext,
    type: v.kind,
    description: null,
  }));
}

export function envVarAuditLogs({
  workspaceId,
  userId,
  context,
  environments,
  keys,
}: {
  workspaceId: string;
  userId: string;
  context: UnkeyAuditLog["context"];
  environments: { id: string; slug: string }[];
  keys: string[];
}): UnkeyAuditLog[] {
  return environments.flatMap((environment) =>
    keys.map((key) => ({
      workspaceId,
      actor: { type: "user", id: userId },
      event: "environment.update",
      description: `Set environment variable ${key} for environment ${environment.id}`,
      resources: [
        { type: "environment", id: environment.id, name: environment.slug, meta: { key } },
      ],
      context,
    })),
  );
}
