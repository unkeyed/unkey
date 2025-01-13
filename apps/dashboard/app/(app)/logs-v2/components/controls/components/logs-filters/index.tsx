import { useFilters } from "@/app/(app)/logs-v2/query-state";
import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { FiltersPopover } from "./components/filters-popover";

export const LogsFilters = () => {
  const { filters } = useFilters();
  return (
    <FiltersPopover>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2",
            filters.length > 0 ? "bg-gray-4" : ""
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
          {filters.length > 0 && (
            <div className="bg-gray-7 rounded size-4 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filters.length}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
