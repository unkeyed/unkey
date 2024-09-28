"use client";

import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { Check, Clock } from "lucide-react";
import { useState } from "react";

const options = [
  {
    value: "1h",
    label: "Last hours",
  },
  {
    value: "3h",
    label: "Last 3 hours",
  },
  {
    value: "6h",
    label: "Last 6 hours",
  },
  {
    value: "12h",
    label: "Last 12 hours",
  },
  {
    value: "24h",
    label: "Last 24 hours",
  },
];

export function HourFilter() {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("1h");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex gap-2 items-center justify-center">
          <Clock className="h-4 w-4" />
          {options.find((o) => o.value === value)?.label}
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0">
        <Command>
          <CommandList>
            <CommandGroup>
              {options.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={(currentValue) => {
                    setValue(currentValue === value ? "" : currentValue);
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value === option.value ? "opacity-100" : "opacity-0"
                    )}
                  />
                  {option.label}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
