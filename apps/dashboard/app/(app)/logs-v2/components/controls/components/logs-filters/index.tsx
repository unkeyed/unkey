import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { FiltersPopover } from "./components/filters-popover";

export const LogsFilters = () => {
  return (
    <FiltersPopover>
      <div className="group">
        <Button
          variant="ghost"
          className="group-data-[state=open]:bg-accent-4"
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
        </Button>
      </div>
    </FiltersPopover>
  );
};
