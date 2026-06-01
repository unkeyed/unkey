"use client";

import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import { cn } from "@/lib/utils";
import { Calendar } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useFilters } from "../../../../hooks/use-filters";

export const IdentitiesDateTime = () => {
  const [title, setTitle] = useState<string | null>(null);
  const { filters, updateFilters } = useFilters();

  useEffect(() => {
    if (!title) {
      setTitle("Last used");
    }
  }, [title]);

  const hasTimeFilters = filters.some((f) =>
    ["lastUsedStart", "lastUsedEnd", "lastUsedSince"].includes(f.field),
  );
  const displayTitle = hasTimeFilters ? (title ?? "Loading...") : "Last used";

  const timeValues: { startTime?: number; endTime?: number; since?: string } = {};
  for (const f of filters) {
    if (f.field === "lastUsedStart") {
      timeValues.startTime = f.value as number;
    } else if (f.field === "lastUsedEnd") {
      timeValues.endTime = f.value as number;
    } else if (f.field === "lastUsedSince") {
      timeValues.since = f.value as string;
    }
  }

  return (
    <DatetimePopover
      maxDate={new Date()}
      initialTimeValues={timeValues}
      onDateTimeChange={(startTime, endTime, since) => {
        const activeFilters = filters.filter(
          (f) => !["lastUsedStart", "lastUsedEnd", "lastUsedSince"].includes(f.field),
        );
        if (since !== undefined) {
          updateFilters([
            ...activeFilters,
            {
              field: "lastUsedSince",
              value: since,
              id: crypto.randomUUID(),
              operator: "is",
            },
          ]);
          return;
        }
        if (startTime) {
          activeFilters.push({
            field: "lastUsedStart",
            value: startTime,
            id: crypto.randomUUID(),
            operator: "is",
          });
          if (endTime) {
            activeFilters.push({
              field: "lastUsedEnd",
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
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 rounded-lg",
            displayTitle === "Loading..." ? "opacity-50" : "",
            hasTimeFilters ? "bg-gray-4" : "",
          )}
          aria-label="Filter identities by last used time"
          aria-haspopup="true"
          title="Press 'T' to toggle time filter"
          disabled={displayTitle === "Loading..."}
        >
          <Calendar className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-[13px]">{displayTitle}</span>
        </Button>
      </div>
    </DatetimePopover>
  );
};
