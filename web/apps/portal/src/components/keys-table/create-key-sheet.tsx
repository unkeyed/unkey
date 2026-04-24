import { CalendarClock, Check, ChevronDown } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "~/components/ui/popover";
import {
  SheetBody,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "~/components/ui/sheet";

export function CreateKeySheet({ appName }: { appName: string }) {
  return (
    <SheetContent className="max-w-3xl">
      <SheetHeader>
        <SheetTitle>Create key</SheetTitle>
        <SheetDescription>Create a new API key for {appName}.</SheetDescription>
      </SheetHeader>

      <SheetBody>
        <div className="flex flex-col gap-6 px-6 py-6">
          <Field label="Name" hint="Helps you identify this key.">
            <Input placeholder="e.g. Production" autoFocus />
          </Field>

          <Field label="Expiration" hint="Leave blank for no expiration.">
            <ExpirationPicker />
          </Field>
        </div>
      </SheetBody>

      <SheetFooter>
        <p className="text-gray-11 text-xs sm:mr-auto">
          Keys are created immediately and ready to use.
        </p>
        <SheetClose asChild>
          <Button variant="ghost">Cancel</Button>
        </SheetClose>
        <Button>Save</Button>
      </SheetFooter>
    </SheetContent>
  );
}

type PresetId = "1d" | "7d" | "30d" | "custom";

const PRESETS: { id: PresetId; label: string; description: string }[] = [
  { id: "1d", label: "1 day", description: "Key expires in 1 day" },
  { id: "7d", label: "1 week", description: "Key expires in 1 week" },
  { id: "30d", label: "1 month", description: "Key expires in 30 days" },
  { id: "custom", label: "Custom", description: "Set custom date and time" },
];

// TODO(@davehawkins): needs more work.
function ExpirationPicker() {
  const [selected, setSelected] = useState<PresetId | null>(null);
  const [custom, setCustom] = useState("");

  const display = useMemo(() => {
    if (!selected) return null;
    if (selected === "custom") return custom ? formatDate(new Date(custom)) : "Custom date";
    const days = selected === "1d" ? 1 : selected === "7d" ? 7 : 30;
    const d = new Date();
    d.setDate(d.getDate() + days);
    return formatDate(d);
  }, [selected, custom]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex h-8 w-full items-center justify-between rounded-md border border-primary/15 bg-background px-3 text-sm shadow-xs transition-colors hover:bg-gray-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1"
        >
          <span className="flex items-center gap-2">
            <CalendarClock className="size-4 text-gray-11" />
            <span className={display ? "text-gray-12" : "text-gray-10"}>
              {display ?? "Select expiration date"}
            </span>
          </span>
          <ChevronDown className="size-4 text-gray-11" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="start">
        <div className="border-primary/10 border-b px-3 py-2">
          <span className="text-[13px] text-gray-11">Choose expiration date</span>
        </div>
        <div className="flex flex-col p-1">
          {PRESETS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => setSelected(p.id)}
              className="flex items-start justify-between gap-4 rounded-sm px-2 py-2 text-left transition-colors hover:bg-gray-2"
            >
              <div className="flex flex-col gap-0.5">
                <span className="font-medium text-gray-12 text-sm">{p.label}</span>
                <span className="text-gray-11 text-xs">{p.description}</span>
              </div>
              {selected === p.id && (
                <Check className="mt-0.5 size-4 shrink-0 text-gray-12" />
              )}
            </button>
          ))}
        </div>
        {selected === "custom" && (
          <div className="border-primary/10 border-t p-3">
            <Input
              type="datetime-local"
              value={custom}
              onChange={(e) => setCustom(e.target.value)}
            />
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

function formatDate(d: Date): string {
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="font-medium text-gray-12 text-sm">{label}</span>
      {children}
      {hint && <span className="text-gray-11 text-xs">{hint}</span>}
    </label>
  );
}
