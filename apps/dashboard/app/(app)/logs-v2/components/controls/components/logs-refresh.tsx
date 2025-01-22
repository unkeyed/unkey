import { trpc } from "@/lib/trpc/client";
import { Refresh3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { toggleLive } = useLogsContext();
  const { filters } = useFilters();
  const { logs } = trpc.useUtils();

  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleSwitch = () => {
    toggleLive(false);
    logs.queryLogs.invalidate();
  };
  return (
    <Button
      onClick={handleSwitch}
      variant="ghost"
      className={cn("px-2 text-accent-12 [&_svg]:text-accent-9")}
      disabled={!hasRelativeFilter}
    >
      <Refresh3 className="size-4 relative z-10" />
      <span className="font-medium text-[13px]">Refresh</span>
    </Button>
  );
};
