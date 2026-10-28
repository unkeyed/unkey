import { ENVIRONMENT_KINDS, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import type { Environment, ListedConnection } from "../connection-rules";

export type ConnectionEnvs = Record<EnvironmentKind, Environment | null>;

export type MergedConnection = {
  targetAppId: string;
  targetAppName: string;
  name: string;
  byEnv: Partial<Record<EnvironmentKind, ListedConnection>>;
};

export function mergeConnections(
  rows: ListedConnection[],
  envs: ConnectionEnvs,
): MergedConnection[] {
  const kindById = new Map(
    ENVIRONMENT_KINDS.flatMap((kind) => {
      const env = envs[kind];
      return env ? [[env.id, kind] as const] : [];
    }),
  );
  const byTarget = new Map<string, MergedConnection>();
  for (const row of rows) {
    const kind = kindById.get(row.environmentId);
    if (!kind) {
      continue;
    }
    const merged = byTarget.get(row.targetAppId) ?? {
      targetAppId: row.targetAppId,
      targetAppName: row.targetAppName,
      name: row.name,
      byEnv: {},
    };
    merged.byEnv[kind] = row;
    merged.name = merged.byEnv.production?.name ?? row.name;
    byTarget.set(row.targetAppId, merged);
  }
  return [...byTarget.values()].sort((a, b) => a.targetAppName.localeCompare(b.targetAppName));
}
