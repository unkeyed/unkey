"use client";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import type { KeyboardEvent, PropsWithChildren } from "react";
import { useEffect, useRef, useState } from "react";
import type { SuggestionOption } from "../logs-datetime.type";

type SuggestionsProps = PropsWithChildren<{
  className?: string;
  options: Array<SuggestionOption>;
  onChange: (id: number) => void;
}>;

export const DateTimeSuggestions = ({ className, options, onChange }: SuggestionsProps) => {
  const [focusedIndex, setFocusedIndex] = useState<number>(
    () => options.findIndex((option) => option.checked) ?? 0,
  );
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);

  useEffect(() => {
    const newCheckedIndex = options.findIndex((option) => option.checked);
    if (newCheckedIndex !== -1) {
      setFocusedIndex(newCheckedIndex);
      itemRefs.current[newCheckedIndex]?.focus();
    }
  }, [options]);

  const handleKeyDown = (e: KeyboardEvent, index: number) => {
    switch (e.key) {
      case "ArrowDown":
      case "j": {
        e.preventDefault();
        const nextIndex = (index + 1) % options.length;
        itemRefs.current[nextIndex]?.focus();
        setFocusedIndex(nextIndex);
        break;
      }
      case "ArrowUp":
      case "k": {
        e.preventDefault();
        const prevIndex = (index - 1 + options.length) % options.length;
        itemRefs.current[prevIndex]?.focus();
        setFocusedIndex(prevIndex);
        break;
      }
      case "Enter":
      case " ":
        e.preventDefault();
        onChange(options[index].id);
        break;
    }
  };

  return (
    <div
      role="radiogroup"
      className={cn("flex flex-col justify-center gap-1.5", className)}
      aria-label="Time range options"
    >
      {options.map(({ id, display, checked }, index) => (
        <div
          key={id}
          className={cn("group relative w-full rounded-lg", "focus-within:outline-none")}
        >
          <button
            type="button"
            role="radio"
            aria-checked={checked}
            ref={(el: HTMLButtonElement | null) => {
              itemRefs.current[index] = el;
            }}
            onClick={() => {
              onChange(id);
              setFocusedIndex(index);
            }}
            onKeyDown={(e) => handleKeyDown(e, index)}
            onFocus={() => setFocusedIndex(index)}
            className={cn(
              "w-full inline-flex items-center justify-between",
              "px-2 py-1.5 rounded-lg",
              "text-[13px] font-medium text-accent-12 text-left",
              "hover:bg-gray-3",
              "focus:outline-none focus:ring-2 focus:ring-accent-7",
              "focus:bg-gray-3",
              checked && "bg-gray-3",
              focusedIndex === index && "bg-gray-3",
            )}
            tabIndex={0}
          >
            <span>{display}</span>
            {checked && (
              <div className="size-4" aria-hidden="true">
                <Check className="text-gray-12/90 size-4" />
              </div>
            )}
          </button>
        </div>
      ))}
    </div>
  );
};
