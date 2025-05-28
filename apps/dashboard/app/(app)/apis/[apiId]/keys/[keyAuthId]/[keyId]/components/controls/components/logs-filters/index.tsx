import { FiltersPopover } from "@/components/logs/checkbox/filters-popover";

import { BarsFilter } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";
import { OutcomesFilter } from "./outcome-filter";

export const LogsFilters = () => {
  const { filters } = useFilters();
  const [open, setOpen] = useState(false);

  return (
    <FiltersPopover
      open={open}
      onOpenChange={setOpen}
      items={[
        {
          id: "outcomes",
          label: "Outcomes",
          shortcut: "o",
          component: <OutcomesFilter onDrawerClose={() => setOpen(false)} />,
        },
      ]}
      activeFilters={filters}
    >
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
            filters.length > 0 ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          size="md"
          title="Press 'F' to toggle filters"
        >
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px] max-md:hidden">Filter</span>
          {filters.length > 0 && (
            <div className="bg-gray-7 rounded h-4 px-1 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
              {filters.length}
            </div>
          )}
        </Button>
      </div>
    </FiltersPopover>
  );
};
