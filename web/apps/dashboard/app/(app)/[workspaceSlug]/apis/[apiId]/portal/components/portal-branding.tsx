"use client";

import { cn } from "@/lib/utils";
import { Input } from "@unkey/ui";

// Mirrors the portal_branding schema columns so this can be wired to trpc later.
// For MVP the customer hosts their own logo image and provides its URL; when
// empty the portal falls back to a plain-text logo using `name`.
export type PortalBrandingValue = {
  name: string;
  logoUrl: string;
  primaryColor: string;
};

const SWATCHES = ["#18181B", "#7C3AED", "#0D9488", "#D97706", "#DC2626"];

export function BrandColorField({
  color,
  onChange,
}: {
  color: string;
  onChange: (color: string) => void;
}) {
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
        style={{ backgroundColor: color }}
      >
        <span className="sr-only">Pick brand color</span>
        <input
          type="color"
          value={color}
          onChange={(e) => onChange(e.target.value.toUpperCase())}
          className="absolute inset-0 cursor-pointer opacity-0"
        />
      </label>
      <Input
        className="w-[96px] font-mono uppercase"
        value={color}
        maxLength={7}
        onChange={(e) => {
          const raw = e.target.value.trim();
          onChange(raw.startsWith("#") ? raw : `#${raw}`);
        }}
      />
    </div>
  );
}
