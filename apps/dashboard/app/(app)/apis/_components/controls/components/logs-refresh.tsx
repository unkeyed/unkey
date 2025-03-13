import { RefreshButton } from "@/components/logs/refresh-button";
import { trpc } from "@/lib/trpc/client";
import { useRouter } from "next/navigation";
import { useFilters } from "../../hooks/use-filters";

export const LogsRefresh = () => {
  const { filters } = useFilters();
  const { api } = trpc.useUtils();
  const { refresh } = useRouter();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    api.overview.timeseries.invalidate();
    refresh();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
