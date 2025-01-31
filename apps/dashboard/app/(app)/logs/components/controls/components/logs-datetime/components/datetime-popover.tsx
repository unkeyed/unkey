"use client";

import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { useKeyboardShortcut } from "@/app/(app)/logs/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { toast } from "@/components/ui/toaster";
import { Button, DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { type PropsWithChildren, useState } from "react";
import { processTimeFilters } from "../utils/process-time";
import { DateTimeSuggestions, type OptionsType } from "./suggestions";

const CUSTOM_OPTION_ID = 10;
const options: OptionsType = [
  {
    id: 1,
    value: "5m",
    display: "Last 5 minutes",
    checked: false,
  },
  {
    id: 2,
    value: "15m",
    display: "Last 15 minutes",
    checked: false,
  },
  {
    id: 3,
    value: "30m",
    display: "Last 30 minutes",
    checked: false,
  },
  {
    id: 4,
    value: "1h",
    display: "Last 1 hour",
    checked: false,
  },
  {
    id: 5,
    value: "3h",
    display: "Last 3 hours",
    checked: false,
  },
  {
    id: 6,
    value: "6h",
    display: "Last 6 hours",
    checked: false,
  },
  {
    id: 7,
    value: "12h",
    display: "Last 12 hours",
    checked: false,
  },
  {
    id: 8,
    value: "24h",
    display: "Last 24 hours",
    checked: false,
  },
  {
    id: 9,
    value: "48h",
    display: "Last 2 days",
    checked: false,
  },
  {
    id: 10,
    value: undefined,
    display: "Custom",
    checked: false,
  },
];

const CUSTOM_DATE_TIME = 10;

interface DatetimePopoverProps extends PropsWithChildren {
  setTitle: (value: string) => void;
  setSelected: (value: boolean) => void;
}

type TimeRangeType = {
  startTime?: number;
  endTime?: number;
};

export const DatetimePopover = ({ children, setTitle, setSelected }: DatetimePopoverProps) => {
  const { filters, updateFilters } = useFilters();

  const sinceFilter = filters.find((f) => f.field === "since");
  const startTimeFilter = filters.find((f) => f.field === "startTime");
  const endTimeFilter = filters.find((f) => f.field === "endTime");

  let initialSuggestions = [...options];

  if (sinceFilter) {
    const matchingSuggestion = initialSuggestions.find((s) => s.value === sinceFilter.value);
    if (matchingSuggestion) {
      initialSuggestions = initialSuggestions.map((s) => ({
        ...s,
        checked: s.id === matchingSuggestion.id,
      }));
      setTitle(matchingSuggestion.display);
      setSelected(true);
    }
  } else if (startTimeFilter) {
    initialSuggestions = initialSuggestions.map((s) => ({
      ...s,
      checked: s.id === CUSTOM_DATE_TIME,
    }));
    setTitle("Custom");
    setSelected(true);
  }

  const [open, setOpen] = useState(false);
  const [suggestions, setSuggestions] = useState<OptionsType>(initialSuggestions);
  const [time, setTime] = useState<TimeRangeType>({
    startTime: startTimeFilter?.value as number | undefined,
    endTime: endTimeFilter?.value as number | undefined,
  });

  useKeyboardShortcut("t", () => {
    setOpen((prev) => !prev);
  });

  const handleSuggestionChange = (id: number) => {
    const tempSuggestions = suggestions.map((suggestion) => {
      const isChecked = suggestion.id === id && !suggestion.checked;
      return {
        ...suggestion,
        checked: isChecked,
      };
    });
    setSuggestions(tempSuggestions);

    const selectedSuggestion = tempSuggestions.find((s) => s.checked)?.value;

    if (id === CUSTOM_OPTION_ID) {
      return;
    }
    const activeFilters = filters.filter((f) => !["since"].includes(f.field));
    if (selectedSuggestion) {
      activeFilters.push({
        field: "since",
        value: selectedSuggestion,
        operator: "is",
        id: crypto.randomUUID(),
      });
    }

    updateFilters(activeFilters);
  };

  const onDateTimeChange = (newRange?: Range, newStart?: TimeUnit, newEnd?: TimeUnit) => {
    const custom = suggestions.find((suggestion) => suggestion.id === CUSTOM_DATE_TIME);
    if (!custom?.checked) {
      handleSuggestionChange(CUSTOM_DATE_TIME);
    }
    const startTimestamp = processTimeFilters(newRange?.from, newStart)?.getTime();
    const endTimestamp = processTimeFilters(newRange?.to, newEnd)?.getTime();
    setTime({
      startTime: startTimestamp,
      endTime: endTimestamp,
    });
  };

  const handleApplyFilter = () => {
    const activeFilters = filters.filter(
      (f) => !["endTime", "startTime", "since"].includes(f.field),
    );
    const selected = suggestions.find((suggestion) => suggestion.checked);
    if (selected) {
      setSelected(true);
      if (selected?.value !== undefined) {
        setTitle(selected.display);
        updateFilters([
          ...activeFilters,
          {
            field: "since",
            value: selected.value,
            id: crypto.randomUUID(),
            operator: "is",
          },
        ]);
        return;
      }
      if (selected?.value === undefined && time.startTime) {
        setTitle("Custom");
        activeFilters.push({
          field: "startTime",
          value: time.startTime,
          id: crypto.randomUUID(),
          operator: "is",
        });
        if (time.endTime) {
          activeFilters.push({
            field: "endTime",
            value: time.endTime,
            id: crypto.randomUUID(),
            operator: "is",
          });
        }
      }
      if (time.startTime === undefined && time.endTime === undefined) {
        toast.error("Please select a date range", {
          duration: 8000,
          important: true,
          position: "top-right",
          style: {
            whiteSpace: "pre-line",
          },
        });
        return;
      }
    } else {
      setSelected(false);
      setTitle("No Filter");
    }
    updateFilters(activeFilters);
    setOpen(false);
  };

  const initialStartDate = filters.find((f) => f.field === "startTime")?.value as
    | number
    | undefined;
  const initialEndDate = filters.find((f) => f.field === "endTime")?.value as number | undefined;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex flex-row items-center">{children}</div>
      </PopoverTrigger>
      <PopoverContent
        className="flex w-full bg-gray-1 dark:bg-black drop-shadow-3 p-0 m-0 border-gray-6 rounded-lg"
        align="start"
      >
        <div className="flex flex-col w-60 px-1.5 py-3 m-0 border-r border-gray-4">
          <PopoverHeader />
          <DateTimeSuggestions options={suggestions} onChange={handleSuggestionChange} />
        </div>
        <DateTime
          onChange={(newRange, newStart, newEnd) => onDateTimeChange(newRange, newStart, newEnd)}
          initialRange={{
            from: initialStartDate ? new Date(initialStartDate) : undefined,
            to: initialEndDate ? new Date(initialEndDate) : undefined,
          }}
          className="gap-3 h-full"
        >
          <DateTime.Calendar
            mode="range"
            className="px-3 pt-2.5 pb-3.5 border-b border-gray-4 text-[13px]"
          />
          <DateTime.TimeInput type="range" className="px-3.5 h-9 mt-0" />
          <DateTime.Actions className="px-3.5 h-full pb-4">
            <Button
              variant="primary"
              className="font-sans w-full h-9 rounded-md"
              onClick={handleApplyFilter}
            >
              Apply Filter
            </Button>
          </DateTime.Actions>
        </DateTime>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full h-8 justify-between px-2">
      <span className="text-gray-9 text-[13px] w-full">Filter by time range</span>
      <KeyboardButton shortcut="T" className="p-0 m-0 min-w-5 w-5 h-5" />
    </div>
  );
};
