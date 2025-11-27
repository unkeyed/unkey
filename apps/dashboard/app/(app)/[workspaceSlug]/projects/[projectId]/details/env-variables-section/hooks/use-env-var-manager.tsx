import { useTRPC } from "@/lib/trpc/client";
import type { Environment } from "../types";

import { useQuery } from "@tanstack/react-query";

type UseEnvVarsManagerProps = {
  projectId: string;
  environment: Environment;
};

export function useEnvVarsManager({ projectId, environment }: UseEnvVarsManagerProps) {
  const trpc = useTRPC();
  const { data } = useQuery(trpc.deploy.environment.list_dummy.queryOptions({ projectId }));

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
