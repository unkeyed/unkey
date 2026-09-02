"use client";

import { useRuntimeLogsFilters } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters";
import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { IconCalendarOutline18 } from "nucleo-ui-outline-18";
import { useEffect, useState } from "react";

export function RuntimeLogsDateTime() {
  const [title, setTitle] = useState<string | null>(null);
  const { filters, updateFilters } = useRuntimeLogsFilters();

  useEffect(() => {
    if (!title) {
      setTitle("Last 6 hours");
    }
  }, [title]);

  const hasTimeFilters = filters.some((f) => ["startTime", "endTime", "since"].includes(f.field));
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : "Last 6 hours";

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
      initialTitle={displayTitle}
      onSuggestionChange={setTitle}
    >
      <Button
        variant="ghost"
        size="md"
        className={cn(
          "data-popup-open:bg-gray-4 px-2 rounded-lg",
          displayTitle === "Loading..." ? "opacity-50" : "",
          displayTitle !== "Last 6 hours" ? "bg-gray-4" : "",
        )}
        aria-label="Filter logs by time"
        aria-haspopup="true"
        title="Press 'T' to toggle filters"
        disabled={displayTitle === "Loading..."}
      >
        <IconCalendarOutline18 className="text-gray-9 size-4" />
        <span className="text-gray-12 font-medium text-[13px]">{displayTitle}</span>
      </Button>
    </DatetimePopover>
  );
}
