import { Check } from "@unkey/icons";
import { Button, Textarea } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useRef, useState } from "react";

type FilterOption<T extends string = string> = {
  id: T;
  label: string;
};

type FilterOperatorInputProps<T extends string> = {
  options: readonly FilterOption<T>[];
  defaultOption?: T;
  defaultText?: string;
  label: string;
  onApply: (selectedId: T, text: string) => void;
};

export const FilterOperatorInput = <T extends string>({
  options,
  defaultOption = options[0].id,
  defaultText = "",
  onApply,
  label,
}: FilterOperatorInputProps<T>) => {
  const [selectedOption, setSelectedOption] = useState<T>(defaultOption);
  const [text, setText] = useState(defaultText);
  const optionRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Initialize the refs array when options change
  useEffect(() => {
    optionRefs.current = optionRefs.current.slice(0, options.length);
  }, [options.length]);

  const handleApply = () => {
    if (text.trim()) {
      onApply(selectedOption, text);
    }
  };

  // Handle keyboard navigation for options
  // INFO: Don't move "stopPropagation" to top. We need "ArrowLeft" to propagate so we can close this popover content when "ArrowLeft" triggered.
  const handleOptionKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
    switch (e.key) {
      case "ArrowDown": {
        e.stopPropagation();
        e.preventDefault();
        // Move to next option or wrap to first
        const nextIndex = (index + 1) % options.length;
        optionRefs.current[nextIndex]?.focus();
        break;
      }

      case "ArrowUp": {
        e.stopPropagation();
        e.preventDefault();
        // Move to previous option or wrap to last
        const prevIndex = (index - 1 + options.length) % options.length;
        optionRefs.current[prevIndex]?.focus();
        break;
      }

      case "Enter":
      case " ":
        e.stopPropagation();
        e.preventDefault();
        setSelectedOption(options[index].id);
        break;

      case "Tab":
        if (!e.shiftKey) {
          e.stopPropagation();
          e.preventDefault();
          textareaRef.current?.focus();
        }
        break;
    }
  };

  const handleTextareaKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleApply();
      return;
    }
  };

  return (
    <div className="flex max-md:flex-col w-full md:w-[500px]">
      <div className="flex flex-col gap-2 p-2 w-full md:w-[180px] md:border-r border-gray-4 items-center">
        {options.map((option, index) => (
          <div
            key={option.id}
            className={cn("group relative w-full rounded-lg", "focus-within:outline-hidden")}
          >
            <button
              type="button"
              id={`option-${option.id}`}
              ref={(el) => {
                optionRefs.current[index] = el;
              }}
              onClick={() => setSelectedOption(option.id)}
              onKeyDown={(e) => handleOptionKeyDown(e, index)}
              aria-selected={selectedOption === option.id}
              className={cn(
                "w-full inline-flex items-center justify-between",
                "px-2 py-1.5 rounded-lg",
                "text-[13px] font-medium text-accent-12 text-left",
                "hover:bg-gray-3",
                "focus:outline-hidden focus:ring-2 focus:ring-accent-7",
                "focus:bg-gray-3",
                selectedOption === option.id && "bg-gray-3",
              )}
            >
              <span>{option.label}</span>
              {selectedOption === option.id && (
                <div className="h-4 w-4" aria-hidden="true">
                  <Check className="text-gray-12/90 h-4 w-4" />
                </div>
              )}
            </button>
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-[14px] py-3 w-full md:w-[320px] px-3">
        <div className="space-y-2">
          <p className="text-gray-9 text-xs" id="filter-operator-title">
            {label}{" "}
            <span className="font-medium text-gray-12">
              {options.find((opt) => opt.id === selectedOption)?.label}
            </span>{" "}
            ...
          </p>
          <Textarea
            ref={textareaRef}
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Enter text"
            onKeyDown={handleTextareaKeyDown}
            className="h-20"
          />
        </div>
        <Button variant="primary" className="py-[14px] w-full h-9 rounded-md" onClick={handleApply}>
          Search
        </Button>
      </div>
    </div>
  );
};
