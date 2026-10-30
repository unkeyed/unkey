import { useQueryTime } from "@/providers/query-time-provider";
import { RefreshButton } from "@unkey/ui";

export const LogsRefresh = () => {
  const { refreshQueryTime } = useQueryTime();

  return <RefreshButton onRefresh={refreshQueryTime} isEnabled />;
};
