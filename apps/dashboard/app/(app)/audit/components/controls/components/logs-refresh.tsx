import { RefreshButton } from "@/components/logs/refresh-button";
import { trpc } from "@/lib/trpc/client";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { filters } = useFilters();
  const { audit } = trpc.useUtils();
  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleRefresh = () => {
    audit.logs.invalidate();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled={Boolean(hasRelativeFilter)} />;
};
