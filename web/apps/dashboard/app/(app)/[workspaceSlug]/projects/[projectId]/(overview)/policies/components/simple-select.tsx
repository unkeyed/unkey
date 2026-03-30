"use client";

import { cn } from "@/lib/utils";

type SimpleSelectProps = {
  label?: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  className?: string;
};

export function SimpleSelect({ label, value, options, onChange, className }: SimpleSelectProps) {
  return (
    <label className={cn("flex flex-col gap-1", className)}>
      {label && <span className="text-xs font-medium text-gray-11">{label}</span>}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-9 w-full rounded-lg border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black text-[13px] text-grayA-12 px-3 py-1.5 focus:border-accent-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden transition-colors"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </label>
  );
}
