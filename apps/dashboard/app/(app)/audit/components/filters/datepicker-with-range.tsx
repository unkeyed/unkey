"use client";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { format, setHours, setMinutes, setSeconds } from "date-fns";
import { ArrowRight, Calendar as CalendarIcon } from "lucide-react";
import { parseAsInteger, useQueryStates } from "nuqs";
import { useEffect, useState } from "react";
import type { DateRange } from "react-day-picker";
import { TimeSplitInput } from "./timesplit-input";

interface DatePickerWithRangeProps extends React.HTMLAttributes<HTMLDivElement> {
  initialParams: {
    startTime: number | null;
    endTime: number | null;
  };
}

export function DatePickerWithRange({ className, initialParams }: DatePickerWithRangeProps) {
  const [interimDate, setInterimDate] = useState<DateRange>({
    from: new Date(),
    to: new Date(),
  });
  const [finalDate, setFinalDate] = useState<DateRange>();
  const [startTime, setStartTime] = useState({ HH: "09", mm: "00", ss: "00" });
  const [endTime, setEndTime] = useState({ HH: "17", mm: "00", ss: "00" });
  const [open, setOpen] = useState(false);

  const [searchParams, setSearchParams] = useQueryStates({
    startTime: parseAsInteger.withDefault(initialParams.startTime ?? 0),
    endTime: parseAsInteger.withDefault(initialParams.endTime ?? 0),
  });

  useEffect(() => {
    if (searchParams.startTime && searchParams.endTime) {
      const from = new Date(searchParams.startTime);
      const to = new Date(searchParams.endTime);
      setFinalDate({ from, to });
      setInterimDate({ from, to });
      setStartTime({
        HH: from.getHours().toString().padStart(2, "0"),
        mm: from.getMinutes().toString().padStart(2, "0"),
        ss: from.getSeconds().toString().padStart(2, "0"),
      });
      setEndTime({
        HH: to.getHours().toString().padStart(2, "0"),
        mm: to.getMinutes().toString().padStart(2, "0"),
        ss: to.getSeconds().toString().padStart(2, "0"),
      });
    }
  }, [searchParams.startTime, searchParams.endTime]);

  const handleFinalDate = (interimDate: DateRange | undefined) => {
    setOpen(false);

    if (interimDate?.from) {
      let mergedFrom = setHours(interimDate.from, Number(startTime.HH));
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
      setSearchParams({
        startTime: mergedFrom.getTime(),
        endTime: mergedTo.getTime(),
      });
    } else {
      setFinalDate(interimDate);
      setSearchParams({
        startTime: null,
        endTime: null,
      });
    }
  };

  return (
    <div className={cn("grid gap-2", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <div
            id="date"
            className={cn(
              "justify-start text-left font-normal flex gap-2 items-center",
              !finalDate && "text-muted-foreground",
            )}
          >
            <div className="flex gap-2 items-center w-fit">
              <div>
                <CalendarIcon className="h-4 w-4" />
              </div>
              {finalDate?.from ? (
                finalDate.to ? (
                  <div className="truncate">
                    {format(finalDate.from, "LLL dd, y")} - {format(finalDate.to, "LLL dd, y")}
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
            disabled={(date) => date > new Date()}
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
                startDate={interimDate.from ?? new Date()}
                endDate={interimDate.to ?? new Date()}
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
                startDate={interimDate.from ?? new Date()}
                endDate={interimDate.to ?? new Date()}
              />
            </div>
            <div className="border-t border-border" />
          </div>
          <div className="flex gap-2 p-2 w-full justify-end bg-background-subtle">
            <Button onClick={() => handleFinalDate(undefined)}>Clear</Button>
            <Button variant="primary" onClick={() => handleFinalDate(interimDate)}>
              Apply
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
