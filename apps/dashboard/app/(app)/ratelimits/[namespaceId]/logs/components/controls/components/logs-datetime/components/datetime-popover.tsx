import { useKeyboardShortcut } from "@/app/(app)/logs/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { processTimeFilters } from "@/lib/utils";
import { Button, DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { type PropsWithChildren, useEffect, useState } from "react";
import type { OptionsType } from "../logs-datetime.type";
import { DateTimeSuggestions } from "./suggestions";

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

interface DatetimePopoverProps extends PropsWithChildren {
  initialTitle: string;
  initialTimeValues: { startTime?: number; endTime?: number; since?: string };
  onSuggestionChange: (title: string) => void;
  onDateTimeChange: (startTime?: number, endTime?: number, since?: string) => void;
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
}: DatetimePopoverProps) => {
  const [open, setOpen] = useState(false);
  useKeyboardShortcut("t", () => setOpen((prev) => !prev));

  const { startTime, since, endTime } = initialTimeValues;
  const [time, setTime] = useState<TimeRangeType>({ startTime, endTime });
  const [suggestions, setSuggestions] = useState<OptionsType>(() => {
    const matchingSuggestion = since
      ? options.find((s) => s.value === since)
      : startTime
        ? options.find((s) => s.id === CUSTOM_OPTION_ID)
        : null;

    return options.map((s) => ({
      ...s,
      checked: s.id === matchingSuggestion?.id,
    }));
  });

  useEffect(() => {
    const newTitle = since
      ? options.find((s) => s.value === since)?.display ?? initialTitle
      : startTime
        ? "Custom"
        : initialTitle;

    onSuggestionChange(newTitle);
  }, [since, startTime, initialTitle, onSuggestionChange]);

  const handleSuggestionChange = (id: number) => {
    if (id === CUSTOM_OPTION_ID) {
      return;
    }

    const newSuggestions = suggestions.map((s) => ({
      ...s,
      checked: s.id === id && !s.checked,
    }));
    setSuggestions(newSuggestions);

    const selectedValue = newSuggestions.find((s) => s.checked)?.value;
    onDateTimeChange(undefined, undefined, selectedValue);
  };

  const handleDateTimeChange = (newRange?: Range, newStart?: TimeUnit, newEnd?: TimeUnit) => {
    setSuggestions(
      suggestions.map((s) => ({
        ...s,
        checked: s.id === CUSTOM_OPTION_ID,
      })),
    );

    setTime({
      startTime: processTimeFilters(newRange?.from, newStart)?.getTime(),
      endTime: processTimeFilters(newRange?.to, newEnd)?.getTime(),
    });
  };

  const handleApplyFilter = () => {
    onDateTimeChange(time.startTime, time.endTime, undefined);
    setOpen(false);
  };

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
          onChange={handleDateTimeChange}
          initialRange={{
            from: startTime ? new Date(startTime) : undefined,
            to: endTime ? new Date(endTime) : undefined,
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
