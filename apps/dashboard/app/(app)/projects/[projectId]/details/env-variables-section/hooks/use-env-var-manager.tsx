import { trpc } from "@/lib/trpc/client";
import type { Environment } from "../types";

type UseEnvVarsManagerProps = {
  projectId: string;
  environment: Environment;
};

export function useEnvVarsManager({ projectId, environment }: UseEnvVarsManagerProps) {
  const { data } = trpc.deploy.project.envs.getEnvs.useQuery({ projectId });
  const envVars = data?.[environment] ?? [];

  // Helper to check for duplicate environment variable keys within the current environment.
  // Used for client-side validation before making server requests.
  const getExistingEnvVar = (key: string, excludeId?: string) => {
    return envVars.find((envVar) => envVar.key.trim() === key.trim() && envVar.id !== excludeId);
  };

  return {
    envVars,
    getExistingEnvVar,
  };
}
