import { trpc } from "@/lib/trpc/client";

export const useFetchUsage = () => {
  const usageQuery = trpc.billing.queryUsage.useQuery(undefined, {
    refetchOnMount: true,
  });

  return { ...usageQuery };
};
