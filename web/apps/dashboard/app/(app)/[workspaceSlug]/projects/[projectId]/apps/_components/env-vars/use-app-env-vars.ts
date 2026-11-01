import { type EnvVar, useEnvVars } from "@/lib/deploy-data";

export type AppEnvVarsLoad =
  | { status: "loading" }
  | { status: "failed"; retry: () => void }
  | { status: "ready"; data: EnvVar[] };

export function useAppEnvVars(projectId: string, appId: string): AppEnvVarsLoad {
  const { data, isLoading, isError, refetch } = useEnvVars(projectId, appId);
  if (isError) {
    return { status: "failed", retry: refetch };
  }
  if (isLoading) {
    return { status: "loading" };
  }
  return { status: "ready", data };
}
