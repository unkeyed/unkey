import { trpc } from "@/lib/trpc/client";

type UseFetchKeyCountProps = {
  apiId: string;
};

export const useFetchKeyCount = ({ apiId }: UseFetchKeyCountProps) => {
  const { data, isLoading, isError } = trpc.api.overview.keyCount.useQuery(
    { apiId },
    {
      enabled: Boolean(apiId),
    },
  );

  return {
    count: data?.count ?? 0,
    isLoading,
    isError,
  };
};
