"use client";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { Check, ChevronDown, CircleCheck } from "@unkey/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import {
  AsYouType,
  type CountryCode,
  getCountries,
  getCountryCallingCode,
} from "libphonenumber-js";
import { useEffect, useMemo, useRef, useState } from "react";

// Country flag emoji from an ISO 3166-1 alpha-2 code (regional indicators),
// so we don't ship a flag asset set.
function flagEmoji(country: string): string {
  return country
    .toUpperCase()
    .replace(/./g, (char) => String.fromCodePoint(127397 + char.charCodeAt(0)));
}

// Native, localized country names — no country-name dependency needed.
const regionNames =
  typeof Intl !== "undefined" && "DisplayNames" in Intl
    ? new Intl.DisplayNames(["en"], { type: "region" })
    : null;

function countryName(country: CountryCode): string {
  try {
    return regionNames?.of(country) ?? country;
  } catch {
    return country;
  }
}

// Countries that have a calling code, sorted alphabetically by display name.
const COUNTRIES: Array<{ code: CountryCode; name: string; callingCode: string }> = getCountries()
  .map((code) => ({
    code,
    name: countryName(code),
    callingCode: getCountryCallingCode(code),
  }))
  .sort((a, b) => a.name.localeCompare(b.name));

// Best-effort default country from the browser locale (e.g. "en-US" → "US").
// Falls back to US, and never returns a code libphonenumber doesn't know.
function detectCountry(): CountryCode {
  if (typeof navigator === "undefined") {
    return "US";
  }
  const locale = navigator.language || navigator.languages?.[0];
  const region = locale?.split("-")[1]?.toUpperCase();
  if (region && getCountries().includes(region as CountryCode)) {
    return region as CountryCode;
  }
  return "US";
}

interface PhoneInputProps {
  /** Called with the E.164 value ("" when empty) and whether it is valid. */
  onChange: (e164: string, valid: boolean) => void;
  disabled?: boolean;
}

export function PhoneInput({ onChange, disabled }: PhoneInputProps) {
  // Start with a stable default for SSR, then refine to the detected country
  // on mount to avoid a hydration mismatch.
  const [country, setCountry] = useState<CountryCode>("US");
  const [nationalValue, setNationalValue] = useState("");
  const [open, setOpen] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setCountry(detectCountry());
  }, []);

  // Derive the E.164 value and validity from the typed national number.
  const { e164, valid } = useMemo(() => {
    const parser = new AsYouType(country);
    parser.input(nationalValue);
    const number = parser.getNumber();
    return { e164: number?.number ?? "", valid: number?.isValid() ?? false };
  }, [country, nationalValue]);

  // Keep the parent in sync with the current value/validity.
  // biome-ignore lint/correctness/useExhaustiveDependencies: onChange identity is caller-stable
  useEffect(() => {
    onChange(valid ? e164 : "", valid);
  }, [e164, valid]);

  const handleInput = (raw: string) => {
    // AsYouType reformats from the full string each keystroke, tolerating
    // edits and deletions of separator characters.
    setNationalValue(new AsYouType(country).input(raw));
  };

  const handleSelectCountry = (next: CountryCode) => {
    setCountry(next);
    setOpen(false);
    // Reformat the existing digits for the newly selected country.
    setNationalValue((current) => new AsYouType(next).input(current));
    inputRef.current?.focus();
  };

  const showInvalid = nationalValue.length > 0 && !valid;

  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm text-white/40">Phone number</span>
      <div
        className={cn(
          "flex items-center h-10 rounded-lg border bg-black transition-colors",
          showInvalid ? "border-[#FB1048]/50" : "border-white/20 focus-within:border-white/50",
          disabled && "opacity-50",
        )}
      >
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger asChild>
            <button
              type="button"
              disabled={disabled}
              aria-label="Select country calling code"
              className="flex items-center gap-1 h-full pl-3 pr-2 text-sm text-white/80 rounded-l-lg hover:bg-white/5 disabled:cursor-not-allowed cursor-pointer shrink-0"
            >
              <span className="text-base leading-none">{flagEmoji(country)}</span>
              <span className="tabular-nums">+{getCountryCallingCode(country)}</span>
              <ChevronDown className="w-3 h-3 text-white/40" />
            </button>
          </PopoverTrigger>
          {/* The popover renders in a portal, and the auth pages theme
              themselves with explicit colors rather than a .dark class, so
              the shared gray tokens don't resolve to dark values here. Style
              the dropdown white-on-black explicitly to match the auth flow. */}
          <PopoverContent className="p-0 w-72 bg-black border-white/10 text-white" align="start">
            <Command
              filter={(value, search) =>
                value.toLowerCase().includes(search.toLowerCase()) ? 1 : 0
              }
              className="bg-black text-white [&_[cmdk-input-wrapper]]:border-white/10 [&_[cmdk-input-wrapper]_svg]:text-white/40"
            >
              <CommandInput
                placeholder="Search country..."
                className="text-white placeholder:text-white/40"
              />
              {/* Thin, dark, overlay-style scrollbar. Without this, desktop
                  browsers (and macOS with "always show scrollbars") render a
                  wide light classic scrollbar over the dark dropdown. */}
              <CommandList className="[scrollbar-width:thin] [scrollbar-color:rgba(255,255,255,0.2)_transparent] [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-white/20 hover:[&::-webkit-scrollbar-thumb]:bg-white/30">
                <CommandEmpty className="text-white/40">No country found.</CommandEmpty>
                <CommandGroup>
                  {COUNTRIES.map((item) => (
                    <CommandItem
                      key={item.code}
                      // Searchable by name, ISO code, and dial code.
                      value={`${item.name} ${item.code} +${item.callingCode}`}
                      onSelect={() => handleSelectCountry(item.code)}
                      className="flex items-center gap-2 cursor-pointer text-white aria-selected:bg-white/10 aria-selected:text-white"
                    >
                      <span className="text-base leading-none">{flagEmoji(item.code)}</span>
                      <span className="flex-1 truncate">{item.name}</span>
                      <span className="text-white/40 tabular-nums">+{item.callingCode}</span>
                      {item.code === country && <Check className="w-4 h-4 text-white shrink-0" />}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>

        <div className="w-px h-5 bg-white/10 shrink-0" />

        <input
          ref={inputRef}
          type="tel"
          inputMode="tel"
          autoComplete="tel-national"
          disabled={disabled}
          value={nationalValue}
          onChange={(e) => handleInput(e.target.value)}
          placeholder="(555) 000-0000"
          className="flex-1 min-w-0 h-full bg-transparent px-3 text-sm text-white placeholder:text-white/30 outline-none disabled:cursor-not-allowed"
        />

        {valid && <CircleCheck className="w-4 h-4 mr-3 text-success-9 shrink-0" />}
      </div>
      {showInvalid && <span className="text-xs text-[#FB1048]">Enter a valid phone number</span>}
    </div>
  );
}
