import { trpc } from "@/lib/trpc/client";
import type { Environment } from "../types";

type UseEnvVarsManagerProps = {
  projectId: string;
  environment: Environment;
};

export function useEnvVarsManager({ projectId, environment }: UseEnvVarsManagerProps) {
  const trpcUtils = trpc.useUtils();

  // Just fetch server data - no optimistic updates needed
  const allEnvVars = trpcUtils.deploy.project.envs.getEnvs.getData({
    projectId,
  });
  const envVars = allEnvVars?.[environment] || [];

  // Helper for validation - each row can check for duplicates
  const getExistingEnvVar = (key: string, excludeId?: string) => {
    return envVars.find((envVar) => envVar.key === key.trim() && envVar.id !== excludeId);
  };

  return {
    envVars,
    getExistingEnvVar,
  };
}
