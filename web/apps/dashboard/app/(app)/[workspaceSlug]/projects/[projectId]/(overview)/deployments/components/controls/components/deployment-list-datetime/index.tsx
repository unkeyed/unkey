import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { cn } from "@/lib/utils";
import { Calendar, ChevronDown } from "@unkey/icons";
import { useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

const TITLE_EMPTY_DEFAULT = "Select Time Range";
export const DeploymentListDatetime = () => {
  const [title, setTitle] = useState<string | null>(TITLE_EMPTY_DEFAULT);
  const { filters, updateFilters } = useFilters();

  const hasTimeFilters = filters.some((f) => ["startTime", "endTime", "since"].includes(f.field));
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : TITLE_EMPTY_DEFAULT;

  const timeValues = filters
    .filter((f) => ["startTime", "endTime", "since"].includes(f.field))
    .reduce(
      (acc, f) => ({
        // biome-ignore lint/performance/noAccumulatingSpread: it's safe to spread
        ...acc,
        [f.field]: f.value,
      }),
      {},
    );

  return (
    <DatetimePopover
      maxDate={new Date()}
      align="end"
      initialTimeValues={timeValues}
      onDateTimeChange={(startTime, endTime, since) => {
        const activeFilters = filters.filter(
          (f) => !["endTime", "startTime", "since"].includes(f.field),
        );
        if (since !== undefined) {
          updateFilters([
            ...activeFilters,
            {
              field: "since",
              value: since,
              id: crypto.randomUUID(),
              operator: "is",
            },
          ]);
          return;
        }
        if (since === undefined && startTime) {
          activeFilters.push({
            field: "startTime",
            value: startTime,
            id: crypto.randomUUID(),
            operator: "is",
          });
          if (endTime) {
            activeFilters.push({
              field: "endTime",
              value: endTime,
              id: crypto.randomUUID(),
              operator: "is",
            });
          }
        }
        updateFilters(activeFilters);
      }}
      initialTitle={displayTitle}
      onSuggestionChange={setTitle}
    >
      <button
        type="button"
        aria-label="Filter logs by time"
        aria-haspopup="true"
        title="Press 'T' to toggle filters"
        disabled={displayTitle === "Loading..."}
        className={cn(
          "flex items-center gap-2 h-9 px-3 w-full",
          "bg-gray-1 border border-grayA-4 rounded-lg",
          "text-[13px] text-accent-12 font-normal",
          "hover:bg-gray-2 transition-colors",
          hasTimeFilters && "bg-gray-2",
          displayTitle === "Loading..." && "opacity-50",
        )}
      >
        <Calendar iconSize="md-medium" className="text-gray-9 shrink-0" />
        <span className="truncate">{displayTitle}</span>
        <ChevronDown className="ml-auto shrink-0" iconSize="md-medium" />
      </button>
    </DatetimePopover>
  );
};
