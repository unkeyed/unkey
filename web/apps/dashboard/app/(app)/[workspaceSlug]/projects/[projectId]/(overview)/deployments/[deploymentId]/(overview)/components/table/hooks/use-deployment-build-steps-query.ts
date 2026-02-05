import { trpc } from "@/lib/trpc/client";
import { useParams } from "next/navigation";

export function useDeploymentBuildStepsQuery() {
  const params = useParams();
  const deploymentId = (params?.deploymentId as string) ?? "";

  const { data, isLoading, error } = trpc.deploy.deployment.buildSteps.useQuery(
    {
      deploymentId,
      includeStepLogs: true,
    },
    {
      enabled: Boolean(deploymentId),
      refetchInterval: 2000,
    },
  );

  return {
    steps: data?.steps ?? [],
    isLoading,
    error,
  };
}
