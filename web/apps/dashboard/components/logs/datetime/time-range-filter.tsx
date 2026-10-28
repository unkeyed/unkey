"use client";

import { DEFAULT_OPTIONS } from "@/components/logs/datetime/constants";
import { DatetimePopover } from "@/components/logs/datetime/datetime-popover";
import {
  type TimeFilter,
  isTimeFilter,
  timeValuesOf,
  withTimeRange,
} from "@/components/logs/datetime/time-filters";
import type { OptionsType } from "@/components/logs/datetime/types";
import type { FilterValue } from "@/components/logs/validation/filter.types";
import { IconCalendarOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";

export type TimeRangeLook = "logs" | "list";

const LOOKS: Record<
  TimeRangeLook,
  { variant: "ghost" | "outline"; ariaLabel: string; labelClassName?: string }
> = {
  logs: { variant: "ghost", ariaLabel: "Filter logs by time", labelClassName: "max-md:hidden" },
  list: { variant: "outline", ariaLabel: "Filter by time range" },
};

export function TimeRangeFilter<Filter extends FilterValue>({
  filters,
  updateFilters,
  look,
  defaultSince = "12h",
  customOptions,
  minDate,
}: {
  filters: Filter[];
  updateFilters: (filters: Array<Filter | TimeFilter>) => void;
  look: TimeRangeLook;
  defaultSince?: string;
  customOptions?: OptionsType;
  minDate?: Date;
}) {
  const defaultTitle =
    DEFAULT_OPTIONS.find((option) => option.value === defaultSince)?.display ?? defaultSince;
  const [title, setTitle] = useState(defaultTitle);
  const { variant, ariaLabel, labelClassName } = LOOKS[look];

  const timeFilters = filters.filter(isTimeFilter);
  const displayTitle = timeFilters.length > 0 ? title : defaultTitle;

  return (
    <DatetimePopover
      // DatetimePopover copies its options into state on mount.
      key={customOptions?.map((option) => option.id).join()}
      maxDate={new Date()}
      minDate={minDate}
      customOptions={customOptions}
      initialTimeValues={timeValuesOf(timeFilters)}
      initialTitle={displayTitle}
      onDateTimeChange={(startTime, endTime, since) =>
        updateFilters(withTimeRange(filters, { startTime, endTime, since }))
      }
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
