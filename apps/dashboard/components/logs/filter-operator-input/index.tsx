import { Textarea } from "@/components/ui/textarea";
import { Check } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";

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

  const handleApply = () => {
    if (text.trim()) {
      onApply(selectedOption, text);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleApply();
    }
  };

  return (
    <div className="flex w-[500px]">
      <div className="flex flex-col gap-2 p-2 w-[180px] border-r border-gray-4 items-center">
        {options.map((option) => (
          <div
            key={option.id}
            className={cn("group relative w-full rounded-lg", "focus-within:outline-none")}
          >
            <button
              type="button"
              onClick={() => setSelectedOption(option.id)}
              className={cn(
                "w-full inline-flex items-center justify-between",
                "px-2 py-1.5 rounded-lg",
                "text-[13px] font-medium text-accent-12 text-left",
                "hover:bg-gray-3",
                "focus:outline-none focus:ring-2 focus:ring-accent-7",
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
      <div className="flex flex-col gap-[14px] py-3 w-[320px] px-3">
        <div className="space-y-2">
          <p className="text-gray-9 text-xs">
            {label}{" "}
            <span className="font-medium text-gray-12">
              {options.find((opt) => opt.id === selectedOption)?.label}
            </span>{" "}
            ...
          </p>
          <Textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Enter text"
            onKeyDown={handleKeyDown}
            className="w-full px-3 py-2 text-sm bg-accent-2 border rounded-lg focus:outline-none focus:ring-4 focus:ring-accent-5 border-accent-12 drop-shadow-sm placeholder:text-gray-8"
          />
        </div>
        <Button variant="primary" className="py-[14px] w-full h-9 rounded-md" onClick={handleApply}>
          Search
        </Button>
      </div>
    </div>
  );
};
