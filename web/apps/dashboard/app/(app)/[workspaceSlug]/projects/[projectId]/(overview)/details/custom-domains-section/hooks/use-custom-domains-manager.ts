"use client";
import { trpc } from "@/lib/trpc/client";

type UseCustomDomainsManagerProps = {
  projectId: string;
};

export function useCustomDomainsManager({ projectId }: UseCustomDomainsManagerProps) {
  const { data, isLoading, error } = trpc.deploy.customDomain.list.useQuery(
    { projectId },
    {
      refetchInterval: (queryData) => {
        const hasPending = queryData?.some(
          (d) => d.verificationStatus === "pending" || d.verificationStatus === "verifying",
        );
        return hasPending ? 5_000 : false;
      },
    },
  );

  const utils = trpc.useUtils();

  const invalidate = () => {
    utils.deploy.customDomain.list.invalidate({ projectId });
  };

  const customDomains = data ?? [];

  const getExistingDomain = (domain: string) => {
    return customDomains.find((d) => d.domain.toLowerCase() === domain.toLowerCase());
  };

  return {
    customDomains,
    isLoading,
    error,
    getExistingDomain,
    invalidate,
  };
}
