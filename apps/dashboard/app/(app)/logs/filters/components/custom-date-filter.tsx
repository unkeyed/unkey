"use client";

import { addHours, format, setHours, setMinutes, setSeconds } from "date-fns";
import type { DateRange } from "react-day-picker";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ArrowRight, Calendar as CalendarIcon } from "lucide-react";
import { useState } from "react";
import TimeSplitInput from "./time-split";

export function DatePickerWithRange({
  className,
}: React.HTMLAttributes<HTMLDivElement>) {
  const [interimDate, setInterimDate] = useState<DateRange>({
    from: new Date(),
    to: new Date(),
  });
  const [finalDate, setFinalDate] = useState<DateRange>();
  const [startTime, setStartTime] = useState({ HH: "09", mm: "00", ss: "00" });
  const [endTime, setEndTime] = useState({ HH: "17", mm: "00", ss: "00" });
  const [calendarOpen, setCalendarOpen] = useState(false);

  const handleFinalDate = (interimDate: DateRange | undefined) => {
    setCalendarOpen(false);

    if (interimDate?.from) {
      let mergedFrom = setHours(interimDate.from, Number(startTime.HH));
      addHours(interimDate?.from, Number(startTime.HH));
      mergedFrom = setMinutes(mergedFrom, Number(startTime.mm));
      mergedFrom = setSeconds(mergedFrom, Number(startTime.ss));

      let mergedTo: Date;
      if (interimDate.to) {
        mergedTo = setHours(interimDate.to, Number(endTime.HH));
        mergedTo = setMinutes(mergedTo, Number(endTime.mm));
        mergedTo = setSeconds(mergedTo, Number(endTime.ss));
      } else {
        mergedTo = setHours(interimDate.from, Number(endTime.HH));
        mergedTo = setMinutes(mergedTo, Number(endTime.mm));
        mergedTo = setSeconds(mergedTo, Number(endTime.ss));
      }

      setFinalDate({ from: mergedFrom, to: mergedTo });
    } else {
      setFinalDate(interimDate);
    }
  };

  return (
    <div className={cn("grid gap-2", className)}>
      <Popover open={calendarOpen} onOpenChange={setCalendarOpen}>
        <PopoverTrigger asChild>
          <div
            id="date"
            className={cn(
              "justify-start text-left font-normal flex gap-2 items-center",
              !finalDate && "text-muted-foreground"
            )}
          >
            <div className="flex gap-2 items-center w-fit">
              <div>
                <CalendarIcon className="h-4 w-4" />
              </div>
              {finalDate?.from ? (
                finalDate.to ? (
                  <div className="truncate">
                    {format(finalDate.from, "LLL dd, y")} -{" "}
                    {format(finalDate.to, "LLL dd, y")}
                  </div>
                ) : (
                  format(finalDate.from, "LLL dd, y")
                )
              ) : (
                <span>Custom</span>
              )}
            </div>
          </div>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0 bg-background">
          <Calendar
            initialFocus
            mode="range"
            defaultMonth={interimDate?.from}
            selected={interimDate}
            onSelect={(date) =>
              setInterimDate({
                from: date?.from,
                to: date?.to,
              })
            }
          />
          <div className="flex flex-col gap-2">
            <div className="border-t border-border" />
            <div className="flex gap-2 items-center w-full justify-evenly">
              <TimeSplitInput
                type="start"
                startTime={startTime}
                endTime={endTime}
                time={startTime}
                setTime={setStartTime}
                setStartTime={setStartTime}
                setEndTime={setEndTime}
                startDate={interimDate.from}
                endDate={interimDate.to}
              />
              <ArrowRight strokeWidth={1.5} size={14} />
              <TimeSplitInput
                type="end"
                startTime={startTime}
                endTime={endTime}
                time={endTime}
                setTime={setEndTime}
                setStartTime={setStartTime}
                setEndTime={setEndTime}
                startDate={interimDate.from}
                endDate={interimDate.to}
              />
            </div>
            <div className="border-t border-border" />
          </div>
          <div className="flex gap-2 p-2 w-full justify-end">
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleFinalDate(undefined)}
            >
              Clear
            </Button>
            <Button
              size="sm"
              variant="primary"
              onClick={() => handleFinalDate(interimDate)}
            >
              Apply
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
