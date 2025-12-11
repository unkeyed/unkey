"use client";

import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { useIsMobile } from "@/hooks/use-mobile";
import { cn, processTimeFilters } from "@/lib/utils";
import { ChevronDown } from "@unkey/icons";
import {
  Button,
  DateTime,
  Drawer,
  KeyboardButton,
  Popover,
  PopoverContent,
  PopoverTrigger,
  type Range,
  type TimeUnit,
} from "@unkey/ui";
import { type PropsWithChildren, type ReactNode, useEffect, useState } from "react";
import { CUSTOM_OPTION_ID, DEFAULT_OPTIONS } from "./constants";
import { DateTimeSuggestions } from "./suggestions";
import type { OptionsType } from "./types";

const CUSTOM_PLACEHOLDER = "Custom";

interface DatetimePopoverProps extends PropsWithChildren {
  initialTitle: string;
  initialTimeValues: { startTime?: number; endTime?: number; since?: string };
  onSuggestionChange: (title: string) => void;
  onDateTimeChange: (startTime?: number, endTime?: number, since?: string) => void;
  customOptions?: OptionsType;
  customHeader?: ReactNode;
  singleDateMode?: boolean; // Props for single date selection
  minDate?: Date; // Props to set a minimum selectable date
  maxDate?: Date; // Props to set a maximum selectable date
}

type TimeRangeType = {
  startTime?: number;
  endTime?: number;
};

export const DatetimePopover = ({
  children,
  initialTitle,
  initialTimeValues,
  onSuggestionChange,
  onDateTimeChange,
  customOptions,
  customHeader,
  singleDateMode = false,
  minDate,
  maxDate,
}: DatetimePopoverProps) => {
  const isMobile = useIsMobile();
  const [timeRangeOpen, setTimeRangeOpen] = useState(false);
  const [open, setOpen] = useState(false);
  useKeyboardShortcut("t", () => setOpen((prev) => !prev));

  const { startTime, since, endTime } = initialTimeValues;
  const [time, setTime] = useState<TimeRangeType>({ startTime, endTime });
  const [lastAppliedTime, setLastAppliedTime] = useState<{
    startTime?: number;
    endTime?: number;
  }>({ startTime, endTime });

  // Use customOptions if provided, otherwise use DEFAULT_OPTIONS
  const OPTIONS = customOptions || DEFAULT_OPTIONS;
  // Find the custom option ID in the provided options
  const CURRENT_CUSTOM_OPTION_ID = customOptions
    ? customOptions.find((o) => o.value === undefined)?.id || CUSTOM_OPTION_ID
    : CUSTOM_OPTION_ID;

  const [suggestions, setSuggestions] = useState<OptionsType>(() => {
    const matchingSuggestion = since
      ? OPTIONS.find((s) => s.value === since)
      : startTime
        ? OPTIONS.find((s) => s.id === CURRENT_CUSTOM_OPTION_ID)
        : null;

    return OPTIONS.map((s) => ({
      ...s,
      checked: s.id === matchingSuggestion?.id,
    }));
  });

  useEffect(() => {
    const newTitle = since
      ? (OPTIONS.find((s) => s.value === since)?.display ?? CUSTOM_PLACEHOLDER)
      : startTime
        ? CUSTOM_PLACEHOLDER
        : initialTitle;

    onSuggestionChange(newTitle);
  }, [since, startTime, initialTitle, onSuggestionChange, OPTIONS]);

  const handleSuggestionChange = (id: number) => {
    if (id === CURRENT_CUSTOM_OPTION_ID) {
      return;
    }

    const newSuggestions = suggestions.map((s) => ({
      ...s,
      checked: s.id === id && !s.checked,
    }));
    setSuggestions(newSuggestions);

    const selectedValue = newSuggestions.find((s) => s.checked)?.value;
    onDateTimeChange(undefined, undefined, selectedValue);
    setOpen(false);
  };

  const handleDateTimeChange = (newRange?: Range, newStart?: TimeUnit, newEnd?: TimeUnit) => {
    setSuggestions(
      suggestions.map((s) => ({
        ...s,
        checked: s.id === CURRENT_CUSTOM_OPTION_ID,
      })),
    );

    // In single date mode, we only care about the "from" date
    // In range mode, we use both from and to dates
    setTime({
      startTime: processTimeFilters(newRange?.from, newStart)?.getTime(),
      endTime: singleDateMode ? undefined : processTimeFilters(newRange?.to, newEnd)?.getTime(),
    });
  };

  const isTimeChanged =
    time.startTime !== lastAppliedTime.startTime ||
    (!singleDateMode && time.endTime !== lastAppliedTime.endTime);

  const handleApplyFilter = () => {
    if (!isTimeChanged) {
      setOpen(false);
      return;
    }

    onDateTimeChange(time.startTime, singleDateMode ? undefined : time.endTime);
    setLastAppliedTime({
      startTime: time.startTime,
      endTime: singleDateMode ? undefined : time.endTime,
    });
    setOpen(false);
  };

  const isDateInRange = (date: Date): boolean => {
    if (minDate && date < minDate) {
      return false;
    }
    if (maxDate && date > maxDate) {
      return false;
    }
    return true;
  };

  const getInitialRange = (): Range => {
    let fromDate = undefined;
    if (startTime) {
      const date = new Date(startTime);
      // Only use if valid, otherwise start clean
      if (isDateInRange(date)) {
        fromDate = date;
      }
    }

    let toDate = undefined;
    if (!singleDateMode && endTime) {
      const date = new Date(endTime);
      if (isDateInRange(date)) {
        toDate = date;
      }
    }

    return { from: fromDate, to: toDate };
  };

  const initialRange = getInitialRange();

  // Create disabled dates array
  const getDisabledDates = () => {
    const disabledDates = [];

    if (minDate) {
      disabledDates.push({ before: minDate });
    }

    if (maxDate) {
      disabledDates.push({ after: maxDate });
    }

    return disabledDates;
  };

  // Common calendar props to ensure consistency between mobile and desktop
  const calendarProps = {
    mode: (singleDateMode ? "single" : "range") as "single" | "range",
    className: "px-3 pt-2.5 pb-3.5 border-b border-gray-4 text-[13px]",
    disabledDates: getDisabledDates(),
    showOutsideDays: true,
  };

  return (
    <>
      {isMobile ? (
        <Drawer.Root open={open} onOpenChange={setOpen}>
          <Drawer.Trigger asChild>
            <div className="flex flex-row items-center">{children}</div>
          </Drawer.Trigger>
          <Drawer.Content className="max-h-[80vh]">
            <div className="flex flex-col w-full gap-2 p-2">
              <button
                type="button"
                onClick={() => setTimeRangeOpen(!timeRangeOpen)}
                className="text-gray-11 h-9 border-border border px-2 text-sm w-full rounded-lg bg-gray-3 flex items-center justify-between"
              >
                <span className="text-gray-9 text-[13px]">
                  {singleDateMode ? "Select a date" : "Filter by time range"}
                </span>
                <ChevronDown
                  className={cn("transition-transform duration-150 ease-out", {
                    "rotate-180": timeRangeOpen,
                  })}
                />
              </button>

              <div className={cn("w-full", !timeRangeOpen && "hidden")}>
                <DateTimeSuggestions options={suggestions} onChange={handleSuggestionChange} />
              </div>
            </div>

            <div className={cn("w-full", timeRangeOpen && "hidden")}>
              <DateTime
                onChange={handleDateTimeChange}
                initialRange={initialRange}
                className="gap-3 h-full w-full flex"
                minDate={minDate}
                maxDate={maxDate}
              >
                <DateTime.Calendar {...calendarProps} />
                <DateTime.TimeInput
                  type={singleDateMode ? "single" : "range"}
                  className="px-3.5 h-9 mt-0"
                />
                <DateTime.Actions className="px-3.5 h-full pb-4">
                  <Button
                    variant="primary"
                    className="font-sans w-full h-9 rounded-md"
                    onClick={handleApplyFilter}
                    disabled={!isTimeChanged}
                  >
                    Apply
                  </Button>
                </DateTime.Actions>
              </DateTime>
            </div>
          </Drawer.Content>
        </Drawer.Root>
      ) : (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <div className="flex flex-row items-center">{children}</div>
          </PopoverTrigger>
          <PopoverContent
            className="flex w-full bg-gray-1 dark:bg-black shadow-2xl p-0 m-0 border-gray-6 rounded-lg"
            align="start"
          >
            <div className="flex flex-col w-60 px-1.5 py-3 m-0 border-r border-gray-4">
              {customHeader || (
                <div className="flex w-full h-8 justify-between px-2">
                  <span className="text-gray-9 text-[13px] w-full">
                    {singleDateMode ? "Select a date" : "Filter by time range"}
                  </span>
                  <KeyboardButton shortcut="T" className="p-0 m-0 min-w-5 w-5 h-5" />
                </div>
              )}
              <DateTimeSuggestions options={suggestions} onChange={handleSuggestionChange} />
            </div>
            <DateTime
              onChange={handleDateTimeChange}
              initialRange={initialRange}
              className="gap-3 h-full"
              minDate={minDate}
              maxDate={maxDate}
            >
              <DateTime.Calendar {...calendarProps} />
              <DateTime.TimeInput
                type={singleDateMode ? "single" : "range"}
                className="px-3.5 h-9 mt-0"
              />
              <DateTime.Actions className="px-3.5 h-full pb-4">
                <Button
                  variant="primary"
                  className="font-sans w-full h-9 rounded-md"
                  onClick={handleApplyFilter}
                  disabled={!isTimeChanged}
                >
                  Apply
                </Button>
              </DateTime.Actions>
            </DateTime>
          </PopoverContent>
        </Popover>
      )}
    </>
  );
};
