import { trpc } from "@/lib/trpc/client";
import { RefreshButton } from "@unkey/ui";
import { useRouter } from "next/navigation";

export const LogsRefresh = () => {
  const { api } = trpc.useUtils();
  const { refresh } = useRouter();

  const handleRefresh = () => {
    api.overview.timeseries.invalidate();
    refresh();
  };

  return <RefreshButton onRefresh={handleRefresh} isEnabled />;
};
