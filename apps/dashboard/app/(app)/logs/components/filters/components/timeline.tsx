"use client";

import { Command, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { sub } from "date-fns";
import { Check, Clock } from "lucide-react";
import { useState } from "react";
import { type Timeline as TimelineType, useLogSearchParams } from "../../../query-state";

const OPTIONS = [
  { value: "1h", label: "Last hour" },
  { value: "3h", label: "Last 3 hours" },
  { value: "6h", label: "Last 6 hours" },
  { value: "12h", label: "Last 12 hours" },
  { value: "24h", label: "Last 24 hours" },
] as const;

export function Timeline() {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState<TimelineType>("1h");

  const { setSearchParams } = useLogSearchParams();

  const handleTimelineSet = (value: TimelineType) => {
    setValue(value);
    const now = new Date();

    const startTime = sub(now, {
      hours: Number.parseInt(value.replace("h", "")),
    });

    setSearchParams({
      startTime: startTime.getTime(),
      endTime: now.getTime(),
    });
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex gap-2 items-center justify-center">
          <Clock className="h-4 w-4" />
          {OPTIONS.find((o) => o.value === value)?.label}
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0">
        <Command>
          <CommandList>
            <CommandGroup>
              {OPTIONS.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={(v) => handleTimelineSet(v as TimelineType)}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value === option.value ? "opacity-100" : "opacity-0",
                    )}
                  />
                  <span className="font-medium">{option.label}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
