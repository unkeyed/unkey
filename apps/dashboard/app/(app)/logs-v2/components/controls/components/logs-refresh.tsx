import { Refresh3 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useLogsContext } from "../../../context/logs";
import { useFilters } from "../../../hooks/use-filters";

export const LogsRefresh = () => {
  const { toggleLive } = useLogsContext();
  const { filters, updateFilters } = useFilters();

  const hasRelativeFilter = filters.find((f) => f.field === "since");

  const handleSwitch = () => {
    toggleLive(false);

    const activeFilters = filters.filter((f) => f.field !== "since");
    if (hasRelativeFilter) {
      updateFilters([
        ...activeFilters,
        {
          field: "since",
          value: hasRelativeFilter.value,
          id: crypto.randomUUID(),
          operator: "is",
        },
      ]);
    }
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
