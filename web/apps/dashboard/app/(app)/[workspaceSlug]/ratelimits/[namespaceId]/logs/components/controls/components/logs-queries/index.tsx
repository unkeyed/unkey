import { QueriesPopover } from "@/components/logs/queries/queries-popover";
import { IconChartBarAxisYOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useFilters } from "../../../../hooks/use-filters";
import { formatFilterValues, getFilterFieldIcon } from "./utils";
export const LogsQueries = () => {
  const { filters, updateFilters } = useFilters();

  return (
    <QueriesPopover
      localStorageName="ratelimitSavedFilters"
      filters={filters}
      updateFilters={updateFilters}
      formatFilterValues={formatFilterValues}
      getFilterFieldIcon={getFilterFieldIcon}
    >
      <Button
        variant="ghost"
        size="md"
        className={cn("data-popup-open:bg-gray-4 px-2 rounded-lg")}
        aria-label="Ratelimit log queries"
        aria-haspopup="true"
        title="Press 'Q' to toggle queries"
      >
        <IconChartBarAxisYOutline18 className="text-gray-9 size-4" />
        <span className="text-gray-12 font-medium text-sm">Queries</span>
      </Button>
    </QueriesPopover>
  );
};
