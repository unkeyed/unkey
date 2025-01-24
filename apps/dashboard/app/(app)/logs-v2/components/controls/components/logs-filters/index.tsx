import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import { FiltersPopover } from "./components/filters-popover";

export const LogsFilters = () => {
  const { filters } = useFilters();
  const [dateFilterCount, setDateFilterCount] = useState<number>(0);
  useEffect(() => {
    const sorted = filters.filter((f) => !["endTime", "startTime", "since"].includes(f.field));
    if (sorted.length > 0) {
      setDateFilterCount(sorted.length);
    } else {
      setDateFilterCount(0);
    }
  }, [filters]);
  return (
    <FiltersPopover>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2",
            dateFilterCount > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {dateFilterCount > 0 && (
            <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {dateFilterCount}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
