import { trpc } from "@/lib/trpc/client";
import type { EnvVar, Environment } from "../types";

type UseEnvVarsManagerProps = {
  projectId: string;
  environment: Environment;
};

export function useEnvVarsManager({ projectId, environment }: UseEnvVarsManagerProps) {
  const { data, isLoading, error } = trpc.deploy.envVar.list.useQuery({ projectId });
  const utils = trpc.useUtils();

  const environmentData = data?.[environment];
  const environmentId = environmentData?.id;
  const envVars: EnvVar[] = environmentData?.variables ?? [];

  // Helper to check for duplicate environment variable keys within the current environment.
  // Used for client-side validation before making server requests.
  const getExistingEnvVar = (key: string, excludeId?: string) => {
    return envVars.find((envVar) => envVar.key.trim() === key.trim() && envVar.id !== excludeId);
  };

  const invalidate = () => {
    utils.deploy.envVar.list.invalidate({ projectId });
  };

  return {
    environmentId,
    envVars,
    getExistingEnvVar,
    isLoading,
    error,
    invalidate,
  };
}
