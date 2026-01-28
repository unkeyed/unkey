import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { cn } from "@/lib/utils";
import { Calendar } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

const TITLE_EMPTY_DEFAULT = "Select Time Range";
export const DeploymentListDatetime = () => {
  const [title, setTitle] = useState<string | null>(TITLE_EMPTY_DEFAULT);
  const { filters, updateFilters } = useFilters();

  useEffect(() => {
    for (const filter of filters) {
      if (["startTime", "endTime", "since"].includes(filter.field)) {
        return;
      }
    }
    setTitle(TITLE_EMPTY_DEFAULT);
  }, [filters]);

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
      initialTitle={title ?? ""}
      onSuggestionChange={setTitle}
    >
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
            title ? "" : "opacity-50",
            title !== TITLE_EMPTY_DEFAULT ? "bg-gray-4" : "",
          )}
          aria-label="Filter logs by time"
          aria-haspopup="true"
          title="Press 'T' to toggle filters"
          disabled={!title}
        >
          <Calendar className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-[13px]">{title ?? "Loading..."}</span>
        </Button>
      </div>
    </DatetimePopover>
  );
};
