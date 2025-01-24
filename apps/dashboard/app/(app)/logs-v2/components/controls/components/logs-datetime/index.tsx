import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { cn } from "@/lib/utils";
import { Calendar } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { DatetimePopover } from "./components/datetime-popover";

export const LogsDateTime = () => {
  const [title, setTitle] = useState<string>("Date Filter");
  const { filters } = useFilters();
  const [dateFilterCount, setDateFilterCount] = useState<boolean>();
  useEffect(() => {
    const sorted = filters.filter((f) => ["endTime", "startTime", "since"].includes(f.field));
    setDateFilterCount(sorted.length > 0);
  }, [filters]);
  return (
    <DatetimePopover setTitle={setTitle}>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2",
            dateFilterCount ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <Calendar className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">{title}</span>
        </Button>
      </div>
    </DatetimePopover>
  );
};
