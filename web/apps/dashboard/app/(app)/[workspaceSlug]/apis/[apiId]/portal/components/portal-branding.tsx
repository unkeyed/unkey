"use client";

import { Input } from "@unkey/ui";
import type { PortalFormValues } from "@/lib/portal/build-update";
import { isHexColor } from "@/lib/portal/validation";
import { cn } from "@/lib/utils";

export type PortalBrandingValue = Pick<PortalFormValues, "logoUrl" | "primaryColor">;

export const DEFAULT_BRAND_COLOR = "#18181B";

const SWATCHES = [DEFAULT_BRAND_COLOR, "#7C3AED", "#0D9488", "#D97706", "#DC2626"];

// Emptying the field must reach the form as "", which `buildPortalUpdate` turns
// into the `null` that clears the column. Re-adding `#` here would make the
// color unclearable.
function normalizeTypedColor(raw: string): string {
  if (raw === "" || raw === "#") {
    return "";
  }
  return raw.startsWith("#") ? raw : `#${raw}`;
}

export function BrandColorField({
  color,
  onChange,
}: {
  color: string;
  onChange: (color: string) => void;
}) {
  // `input[type=color]` only accepts a full hex, so it falls back rather than
  // going uncontrolled while the text field is empty or mid-edit.
  const pickerColor = isHexColor(color) ? color : DEFAULT_BRAND_COLOR;

  return (
    <div className="flex items-center gap-2">
      <div className="flex items-center gap-1.5 pr-1">
        {SWATCHES.map((hex) => (
          <button
            key={hex}
            type="button"
            aria-label={`Use ${hex}`}
            onClick={() => onChange(hex)}
            className={cn(
              "size-5 rounded-full border border-grayA-6 transition-shadow",
              color.toUpperCase() === hex &&
                "ring-2 ring-accent-12 ring-offset-2 ring-offset-gray-1",
            )}
            style={{ backgroundColor: hex }}
          />
        ))}
      </div>
      <label
        className="relative size-9 shrink-0 cursor-pointer overflow-hidden rounded-lg border border-gray-5"
        style={{ backgroundColor: pickerColor }}
      >
        <span className="sr-only">Pick brand color</span>
        <input
          type="color"
          value={pickerColor}
          onChange={(e) => onChange(e.target.value.toUpperCase())}
          className="absolute inset-0 cursor-pointer opacity-0"
        />
      </label>
      <Input
        aria-label="Primary color"
        className="w-[96px] font-mono uppercase"
        value={color}
        placeholder={DEFAULT_BRAND_COLOR}
        maxLength={7}
        onChange={(e) => onChange(normalizeTypedColor(e.target.value.trim()))}
      />
    </div>
  );
}
