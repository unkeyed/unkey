import { useEffect, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";

type PresetId = "none" | "1h" | "1d" | "7d" | "30d" | "90d" | "180d" | "365d";

const PRESETS: { id: PresetId; label: string; offsetMs: number | null }[] = [
  { id: "none", label: "No expiration", offsetMs: null },
  { id: "1h", label: "1 hour", offsetMs: 60 * 60 * 1000 },
  { id: "1d", label: "1 day", offsetMs: 24 * 60 * 60 * 1000 },
  { id: "7d", label: "1 week", offsetMs: 7 * 24 * 60 * 60 * 1000 },
  { id: "30d", label: "1 month", offsetMs: 30 * 24 * 60 * 60 * 1000 },
  { id: "90d", label: "3 months", offsetMs: 90 * 24 * 60 * 60 * 1000 },
  { id: "180d", label: "6 months", offsetMs: 180 * 24 * 60 * 60 * 1000 },
  { id: "365d", label: "1 year", offsetMs: 365 * 24 * 60 * 60 * 1000 },
];

type ExpirationPickerProps = {
  id?: string;
  value: Date | undefined;
  onChange: (value: Date | undefined) => void;
  invalid?: boolean;
};

export function ExpirationPicker({ id, value, onChange, invalid }: ExpirationPickerProps) {
  // Edit flow opens with an existing absolute date that won't match any
  // preset; selectedId stays undefined and the formatted date shows via the
  // SelectValue placeholder.
  const [selectedId, setSelectedId] = useState<PresetId | undefined>(() =>
    value === undefined ? "none" : undefined,
  );

  useEffect(() => {
    if (value === undefined && selectedId !== "none") {
      setSelectedId("none");
    }
  }, [value, selectedId]);

  const handleChange = (next: string) => {
    const preset = PRESETS.find((p) => p.id === next);
    if (!preset) {
      return;
    }
    setSelectedId(preset.id);
    onChange(preset.offsetMs === null ? undefined : new Date(Date.now() + preset.offsetMs));
  };

  return (
    <Select value={selectedId} onValueChange={handleChange}>
      <SelectTrigger id={id} aria-invalid={invalid}>
        <SelectValue placeholder={value ? formatDate(value) : "No expiration"} />
      </SelectTrigger>
      <SelectContent>
        {PRESETS.map((p) => (
          <SelectItem key={p.id} value={p.id}>
            {p.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
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
