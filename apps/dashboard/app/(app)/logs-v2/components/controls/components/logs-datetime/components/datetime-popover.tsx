import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button, DateTime, type Range, type TimeUnit } from "@unkey/ui";
import { type PropsWithChildren, useState } from "react";
import { DateTimeSuggestions, type OptionsType } from "./suggestions";

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
  setTitle: (value: string) => void;
}

export const DatetimePopover = ({ children, setTitle }: DatetimePopoverProps) => {
  const [open, setOpen] = useState(false);
  const [suggestions, setSuggestions] = useState<OptionsType>(options);
  const { filters, updateFilters } = useFilters();
  const [startEpoch, setStartEpoch] = useState<number | undefined>();
  const [endEpoch, setEndEpoch] = useState<number | undefined>();
  useKeyboardShortcut("t", () => {
    setOpen((prev) => !prev);
  });

  //Process new Date and time filters to be added to the filters as time since epoch
  const processTimeFilters = (date?: Date, newTime?: TimeUnit) => {
    if (date) {
      const hours = newTime?.HH ? Number.parseInt(newTime.HH) : 0;
      const minutes = newTime?.mm ? Number.parseInt(newTime.mm) : 0;
      const seconds = newTime?.ss ? Number.parseInt(newTime.ss) : 0;
      date.setHours(hours, minutes, seconds, 0);
      return date;
    }
  };

  const handleSuggestionChange = (id: number) => {
    const tempSuggestions = suggestions.map((suggestion) => {
      return {
        ...suggestion,
        checked: suggestion.id === id,
      };
    });

    setSuggestions(tempSuggestions);
  };
  const onDateTimeChange = (newRange?: Range, newStart?: TimeUnit, newEnd?: TimeUnit) => {
    handleSuggestionChange(10); //Custom when selecting something in datetime
    const startTimestamp = processTimeFilters(newRange?.from, newStart)?.getTime();
    const endTimestamp = processTimeFilters(newRange?.to ?? newRange?.from, newEnd)?.getTime();
    if (newRange?.from) {
      setStartEpoch(startTimestamp);
    } else {
      setStartEpoch(undefined);
    }
    if (newRange?.to) {
      setEndEpoch(endTimestamp);
    } else {
      setEndEpoch(undefined);
    }
  };

  const handleApplyFilter = () => {
    setOpen(false);
    const activeFilters = filters.filter(
      (f) => !["endTime", "startTime", "since"].includes(f.field),
    );
    const selected = suggestions.find((suggestion) => suggestion.checked);
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
    if (selected?.value === undefined && startEpoch) {
      setTitle("Custom");
      activeFilters.push({
        field: "startTime",
        value: startEpoch,
        id: crypto.randomUUID(),
        operator: "is",
      });

      if (endEpoch) {
        activeFilters.push({
          field: "endTime",
          value: endEpoch,
          id: crypto.randomUUID(),
          operator: "is",
        });
      }
    }
    updateFilters(activeFilters);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex flex-row items-center">{children}</div>
      </PopoverTrigger>
      <PopoverContent
        className="flex w-fit bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg p-2 gap-2"
        side="right"
        align="start"
        sideOffset={12}
        tabIndex={-1}
      >
        <div className="flex flex-col">
          <PopoverHeader />
          <DateTimeSuggestions
            options={suggestions}
            onChange={(id: number) => handleSuggestionChange(id)}
          />
        </div>
        <div className="flex flex-col mt-2">
          <DateTime
            onChange={(newRange, newStart, newEnd) => onDateTimeChange(newRange, newStart, newEnd)}
          >
            <DateTime.Calendar mode="range" />
            <DateTime.TimeInput type="range" />
            <DateTime.Actions>
              <Button className="w-full" variant="primary" onClick={handleApplyFilter}>
                Apply
              </Button>
            </DateTime.Actions>
          </DateTime>
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1 gap-4">
      <span className="text-gray-9 text-[13px]">Filter by time range</span>
      <KeyboardButton shortcut="T" />
    </div>
  );
};
