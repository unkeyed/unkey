"use client";

import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import type { OptionsType } from "@/components/logs/datetime/types";
import type { FilterValue } from "@/components/logs/validation/filter.types";
import { IconCalendarOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";

type TimeField = "startTime" | "endTime" | "since";
type TimeFilter =
  | FilterValue<"startTime", "is", number>
  | FilterValue<"endTime", "is", number>
  | FilterValue<"since", "is", string>;

export type TimeRangeLook = "logs" | "logsCompact" | "list";

const TIME_FIELDS: readonly TimeField[] = ["startTime", "endTime", "since"];
const DEFAULT_TITLE = "Last 12 hours";

const LOOKS: Record<
  TimeRangeLook,
  { variant: "ghost" | "outline"; ariaLabel: string; labelClassName?: string }
> = {
  logs: { variant: "ghost", ariaLabel: "Filter logs by time" },
  logsCompact: {
    variant: "ghost",
    ariaLabel: "Filter logs by time",
    labelClassName: "max-md:hidden",
  },
  list: { variant: "outline", ariaLabel: "Filter by time range" },
};

function isTimeFilter<Filter extends FilterValue>(
  filter: Filter,
): filter is Filter & { field: TimeField } {
  return TIME_FIELDS.some((field) => field === filter.field);
}

export function TimeRangeFilter<Filter extends FilterValue>({
  filters,
  updateFilters,
  look,
  defaultTitle = DEFAULT_TITLE,
  customOptions,
  minDate,
}: {
  filters: Filter[];
  updateFilters: (filters: Array<Filter | TimeFilter>) => void;
  look: TimeRangeLook;
  defaultTitle?: string;
  customOptions?: OptionsType;
  minDate?: Date;
}) {
  const [title, setTitle] = useState(defaultTitle);
  const { variant, ariaLabel, labelClassName } = LOOKS[look];

  const timeFilters = filters.filter(isTimeFilter);
  const displayTitle = timeFilters.length > 0 ? title : defaultTitle;
  const timeValues = Object.fromEntries(timeFilters.map((f) => [f.field, f.value]));

  return (
    <DatetimePopover
      maxDate={new Date()}
      minDate={minDate}
      customOptions={customOptions}
      initialTimeValues={timeValues}
      initialTitle={displayTitle}
      onDateTimeChange={(startTime, endTime, since) => {
        const activeFilters: Array<Filter | TimeFilter> = filters.filter((f) => !isTimeFilter(f));
        if (since !== undefined) {
          activeFilters.push({
            field: "since",
            value: since,
            id: crypto.randomUUID(),
            operator: "is",
          });
        } else if (startTime) {
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
      onSuggestionChange={setTitle}
    >
      <Button
        variant={variant}
        size="md"
        className={cn(
          "data-popup-open:bg-gray-4 px-2 rounded-lg",
          displayTitle !== defaultTitle ? "bg-gray-4" : "",
        )}
        aria-label={ariaLabel}
        aria-haspopup="dialog"
        aria-keyshortcuts="T"
        title="Press 'T' to toggle filters"
      >
        <IconCalendarOutline18 className="text-gray-9 size-4" />
        <span className={cn("text-gray-12 font-medium text-sm", labelClassName)}>
          {displayTitle}
        </span>
      </Button>
    </DatetimePopover>
  );
}
