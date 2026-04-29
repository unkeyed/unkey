import { Calendar as CalendarIcon, Check, ChevronDown, Clock } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "~/components/ui/button";
import { Calendar } from "~/components/ui/calendar";
import { Input } from "~/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "~/components/ui/popover";
import { cn } from "~/lib/utils";

type PresetId = "1h" | "1d" | "7d" | "30d";

const PRESETS: { id: PresetId; label: string; addOffset: (d: Date) => void }[] = [
  { id: "1h", label: "1 hour", addOffset: (d) => d.setHours(d.getHours() + 1) },
  { id: "1d", label: "1 day", addOffset: (d) => d.setDate(d.getDate() + 1) },
  { id: "7d", label: "1 week", addOffset: (d) => d.setDate(d.getDate() + 7) },
  { id: "30d", label: "1 month", addOffset: (d) => d.setDate(d.getDate() + 30) },
];

type ExpirationPickerProps = {
  id?: string;
  value: Date | undefined;
  onChange: (value: Date | undefined) => void;
  invalid?: boolean;
};

export function ExpirationPicker({ id, value, onChange, invalid }: ExpirationPickerProps) {
  const [open, setOpen] = useState(false);
  const [presetId, setPresetId] = useState<PresetId | null>(null);
  const [pendingDay, setPendingDay] = useState<Date | undefined>(undefined);
  const [pendingTime, setPendingTime] = useState("23:59:59");

  useEffect(() => {
    if (!open) {
      return;
    }
    setPendingDay(value);
    setPendingTime(value ? toTimeString(value) : "23:59:59");
  }, [open, value]);

  const display = value ? formatDate(value) : null;

  const applyPreset = (preset: (typeof PRESETS)[number]) => {
    const d = new Date();
    preset.addOffset(d);
    onChange(d);
    setPresetId(preset.id);
    setOpen(false);
  };

  const applyCustom = () => {
    if (!pendingDay) {
      return;
    }
    onChange(combine(pendingDay, pendingTime));
    setOpen(false);
  };

  const canApply = !!pendingDay && !(value && sameSecond(combine(pendingDay, pendingTime), value));

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          id={id}
          variant="outline"
          aria-invalid={invalid}
          className="w-full justify-between px-3 font-normal aria-invalid:border-error-9 aria-invalid:focus-visible:ring-error-9"
        >
          <span className="flex items-center gap-2">
            <CalendarIcon className="-ml-1 size-4 text-gray-11" />
            <span className={display ? "text-gray-12" : "text-gray-10"}>
              {display ?? "Select expiration date"}
            </span>
          </span>
          <ChevronDown className="size-4 text-gray-11" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="flex w-auto rounded-lg border-primary/15 bg-background p-0 shadow-2xl"
        align="end"
      >
        <div className="flex w-44 flex-col border-primary/10 border-r p-1.5">
          <div className="px-2 pt-1 pb-2 text-[13px] text-gray-11">Expiration</div>
          {PRESETS.map((p) => (
            <Button
              key={p.id}
              variant="ghost"
              onClick={() => applyPreset(p)}
              className={cn("w-full justify-between ", presetId === p.id && "bg-gray-3")}
            >
              <span>{p.label}</span>
              {presetId === p.id && <Check />}
            </Button>
          ))}
        </div>
        <div className="flex flex-col gap-3 p-3">
          <Calendar
            selected={pendingDay}
            onSelect={(day) => {
              setPendingDay(day);
              setPresetId(null);
            }}
            minDate={startOfToday()}
          />
          <div className="-mx-3 flex items-center gap-2 border-primary/10 border-t px-4 pt-3">
            <div className="relative">
              <Clock className="-translate-y-1/2 absolute top-1/2 left-2.5 size-3.5 text-gray-11" />
              <Input
                type="text"
                value={pendingTime}
                onChange={(e) => {
                  setPendingTime(e.target.value);
                  setPresetId(null);
                }}
                placeholder="HH:MM:SS"
                aria-label="Expiration time"
                className="w-28 pl-7 tabular-nums"
              />
            </div>
            <Button onClick={applyCustom} disabled={!canApply} className="ml-auto">
              Apply
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function formatDate(d: Date): string {
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

function toTimeString(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function combine(day: Date, time: string): Date {
  const [hh, mm, ss] = time.split(":");
  const d = new Date(day);
  d.setHours(
    Number.parseInt(hh ?? "0", 10) || 0,
    Number.parseInt(mm ?? "0", 10) || 0,
    Number.parseInt(ss ?? "0", 10) || 0,
    0,
  );
  return d;
}

function sameSecond(a: Date, b: Date): boolean {
  return Math.floor(a.getTime() / 1000) === Math.floor(b.getTime() / 1000);
}

function startOfToday(): Date {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d;
}
