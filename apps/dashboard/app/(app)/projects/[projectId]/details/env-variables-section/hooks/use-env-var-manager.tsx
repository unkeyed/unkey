import { trpc } from "@/lib/trpc/client";
import type { Environment } from "../types";

type UseEnvVarsManagerProps = {
  projectId: string;
  environment: Environment;
};

export function useEnvVarsManager({ projectId, environment }: UseEnvVarsManagerProps) {
  const trpcUtils = trpc.useUtils();

  const allEnvVars = trpcUtils.deploy.project.envs.getEnvs.getData({
    projectId,
  });
  const envVars = allEnvVars?.[environment] || [];

  // Helper to check for duplicate environment variable keys within the current environment.
  // Used for client-side validation before making server requests.
  const getExistingEnvVar = (key: string, excludeId?: string) => {
    return envVars.find((envVar) => envVar.key === key.trim() && envVar.id !== excludeId);
  };

  return {
    envVars,
    getExistingEnvVar,
  };
}
