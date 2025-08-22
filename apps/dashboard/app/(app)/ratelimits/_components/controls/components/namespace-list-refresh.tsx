import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useNamespaceListFilters } from "../../hooks/use-namespace-list-filters";

export const NamespaceListRefresh = () => {
  const { filters } = useNamespaceListFilters();
  const { ratelimit } = trpc.useUtils();
  const { refresh } = useRouter();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    ratelimit.logs.queryRatelimitTimeseries.invalidate();
    refresh();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
