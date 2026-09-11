import { Calendar, Check, ChevronDown } from "lucide-react";
import { useState } from "react";
import { Button } from "~/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "~/components/ui/popover";
import { cn } from "~/lib/utils";
import type { TimePreset, TimePresetId } from "./time-presets";

type TimeFilterProps = {
  presets: TimePreset[];
  value: TimePreset;
  isDefault: boolean;
  onChange: (id: TimePresetId) => void;
};

export function TimeFilter({ presets, value, isDefault, onChange }: TimeFilterProps) {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            className={cn("gap-2", !isDefault && "bg-gray-2")}
            aria-label="Filter by time"
          >
            <Calendar className="text-gray-10" />
            <span className="truncate">{value.label}</span>
            <ChevronDown className="text-gray-10" />
          </Button>
        }
      />
      <PopoverContent className="w-48 p-1" align="end">
        <fieldset aria-label="Presets">
          {presets.map((preset) => {
            const active = preset.id === value.id;
            return (
              <button
                key={preset.id}
                type="button"
                aria-pressed={active}
                onClick={() => {
                  onChange(preset.id);
                  setOpen(false);
                }}
                className={cn(
                  "flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-gray-3 focus-visible:bg-gray-3 focus-visible:outline-none",
                  active ? "font-medium text-gray-12" : "text-gray-11",
                )}
              >
                {preset.label}
                {active && <Check className="size-3.5" aria-hidden="true" />}
              </button>
            );
          })}
        </fieldset>
      </PopoverContent>
    </Popover>
  );
}
