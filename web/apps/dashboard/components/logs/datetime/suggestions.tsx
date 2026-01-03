import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { Check } from "@unkey/icons";
import type { KeyboardEvent, PropsWithChildren } from "react";
import { useEffect, useRef, useState } from "react";
import type { SuggestionOption } from "./types";

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
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const newCheckedIndex = options.findIndex((option) => option.checked);
    if (newCheckedIndex !== -1) {
      setFocusedIndex(newCheckedIndex);
      itemRefs.current[newCheckedIndex]?.focus();
    }
  }, [options]);

  const scrollIntoView = (index: number) => {
    const element = itemRefs.current[index];
    if (element && scrollAreaRef.current) {
      const container = scrollAreaRef.current;
      const elementRect = element.getBoundingClientRect();
      const containerRect = container.getBoundingClientRect();

      if (elementRect.bottom > containerRect.bottom) {
        container.scrollTop += elementRect.bottom - containerRect.bottom;
      } else if (elementRect.top < containerRect.top) {
        container.scrollTop += elementRect.top - containerRect.top;
      }
    }
  };

  const handleKeyDown = (e: KeyboardEvent, index: number) => {
    switch (e.key) {
      case "ArrowDown":
      case "j": {
        e.preventDefault();
        const nextIndex = (index + 1) % options.length;
        itemRefs.current[nextIndex]?.focus();
        setFocusedIndex(nextIndex);
        scrollIntoView(nextIndex);
        break;
      }
      case "ArrowUp":
      case "k": {
        e.preventDefault();
        const prevIndex = (index - 1 + options.length) % options.length;
        itemRefs.current[prevIndex]?.focus();
        setFocusedIndex(prevIndex);
        scrollIntoView(prevIndex);
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
      className={cn("flex flex-col justify-center w-full", className)}
      aria-label="Time range options"
    >
      <ScrollArea className="w-full rounded-md md:max-h-[380px]" ref={scrollAreaRef}>
        <div className="flex flex-col gap-1.5 p-1">
          {options.map(({ id, display, checked }, index) => (
            <div
              key={id}
              className={cn("group relative w-full rounded-lg", "focus-within:outline-none")}
            >
              <button
                type="button"
                // biome-ignore lint/a11y/useSemanticElements: its okay
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
      </ScrollArea>
    </div>
  );
};
