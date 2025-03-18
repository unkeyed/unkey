import { trpc } from "@/lib/trpc/client"

export const useFetchUsage = () => {
  const usageQuery = trpc.billing.queryUsage.useQuery(undefined, {
    refetchOnWindowFocus: true,
    refetchOnMount: true,
    refetchOnReconnect: true,
    refetchInterval: 1000 * 60 * 3,
  })

  return { ...usageQuery }
}
