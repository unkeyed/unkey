"use client";

import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { cn } from "@/lib/utils";
import { Calendar } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

export function RuntimeLogsDateTime() {
  const [title, setTitle] = useState<string | null>(null);
  const { filters, updateFilters } = useRuntimeLogsFilters();

  useEffect(() => {
    if (!title) {
      setTitle("Last 6 hours");
    }
  }, [title]);

  const timeValues = filters
    .filter((f) => ["startTime", "endTime", "since"].includes(f.field))
    .reduce(
      (acc, f) => {
        acc[f.field] = f.value;
        return acc;
      },
      {} as Record<string, string | number>,
    );

  return (
    <DatetimePopover
      maxDate={new Date()}
      initialTimeValues={timeValues}
      onDateTimeChange={(startTime, endTime, since) => {
        const nonTimeFilters = filters.filter(
          (f) => !["since", "startTime", "endTime"].includes(f.field),
        );
        const newFilters = [...nonTimeFilters];

        if (since !== undefined) {
          newFilters.push({
            id: crypto.randomUUID(),
            field: "since" as const,
            operator: "is" as const,
            value: since,
          });
        } else if (startTime) {
          newFilters.push({
            id: crypto.randomUUID(),
            field: "startTime" as const,
            operator: "is" as const,
            value: startTime,
          });
          if (endTime) {
            newFilters.push({
              id: crypto.randomUUID(),
              field: "endTime" as const,
              operator: "is" as const,
              value: endTime,
            });
          }
        }

        updateFilters(newFilters);
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
            title !== "Last 6 hours" ? "bg-gray-4" : "",
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
}
