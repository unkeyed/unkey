"use client";

import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { useRef, useState } from "react";
import type { KeyboardEvent, PropsWithChildren } from "react";

type SuggestionOption = {
  id: number;
  value: string | number | undefined;
  display: string;
  checked: boolean;
};

export type OptionsType = SuggestionOption[];

interface SuggestionsProps extends PropsWithChildren {
  className?: string;
  options: Array<SuggestionOption>;
  onChange: (id: number) => void;
}

export const DateTimeSuggestions = ({ className, options, onChange }: SuggestionsProps) => {
  // Set initial focused index to checked item or first item
  const initialIndex = options.findIndex((option) => option.checked) ?? 0;
  const [focusedIndex, setFocusedIndex] = useState<number>(initialIndex);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const handleKeyDown = (e: KeyboardEvent) => {
    switch (e.key) {
      case "ArrowDown":
      case "j":
        e.preventDefault();
        setFocusedIndex((prev) => {
          const newIndex = (prev + 1) % options.length;
          itemRefs.current[newIndex]?.focus();
          return newIndex;
        });
        break;
      case "ArrowUp":
      case "k":
        e.preventDefault();
        setFocusedIndex((prev) => {
          const newIndex = (prev - 1 + options.length) % options.length;
          itemRefs.current[newIndex]?.focus();
          return newIndex;
        });
        break;
      case "Enter":
      case " ":
        e.preventDefault();
        onChange(options[focusedIndex].id);
        break;
    }
  };

  return (
    <div
      className={cn("flex flex-col justify-center gap-1.5", className)}
      onKeyDown={handleKeyDown}
      role="listbox"
      aria-label="Time range options"
    >
      {options.map(({ id, display, checked }, index) => (
        <div
          key={id}
          className={cn("group relative w-full rounded-lg", "focus-within:outline-none")}
          role="presentation"
        >
          <button
            type="button"
            ref={(el: HTMLButtonElement | null) => {
              itemRefs.current[index] = el;
            }}
            onClick={() => onChange(id)}
            onFocus={() => setFocusedIndex(index)}
            className={cn(
              "w-full inline-flex items-center justify-between",
              "px-2 py-1.5 rounded-lg",
              "text-[13px] font-medium text-accent-12 text-left",
              "hover:bg-gray-3",
              "focus:outline-none",
              "focus:bg-gray-3",
              checked && "bg-gray-3",
              focusedIndex === index && "bg-gray-3",
            )}
            role="option"
            aria-selected={checked}
            tabIndex={index === focusedIndex ? 0 : -1}
          >
            <span>{display}</span>
            {checked && (
              <div className="size-4">
                <Check className="text-gray-12/90 size-4" />
              </div>
            )}
          </button>
        </div>
      ))}
    </div>
  );
};
