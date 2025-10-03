import { QueriesPopover } from "@/components/logs/queries/queries-popover";
import { cn } from "@/lib/utils";
import { ChartBarAxisY } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";
import { formatFilterValues, getFilterFieldIcon, shouldTruncateRow } from "./utils";
export const LogsQueries = () => {
  const { filters, updateFilters } = useFilters();

  return (
    <QueriesPopover
      localStorageName="auditSavedFilters"
      filters={filters}
      updateFilters={updateFilters}
      formatFilterValues={formatFilterValues}
      getFilterFieldIcon={getFilterFieldIcon}
      shouldTruncateRow={shouldTruncateRow}
    >
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn("group-data-[state=open]:bg-gray-4 px-2 rounded-lg")}
          aria-label="Audit log queries"
          aria-haspopup="true"
          title="Press 'Q' to toggle queries"
        >
          <ChartBarAxisY iconsize="md-medium" className="mt-1 ml-[3px] text-gray-9" />
          <span className="text-gray-12 font-medium text-[13px] leading-4">Queries</span>
        </Button>
      </div>
    </QueriesPopover>
  );
};
